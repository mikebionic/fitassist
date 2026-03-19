package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mike/fitassist/internal/ai"
	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/repository"
)

type NotificationService struct {
	notifRepo    *repository.NotificationRepository
	healthRepo   *repository.HealthRepository
	telegramRepo *repository.TelegramRepository
	aiClient     *ai.Client
	sendMsg      func(ctx context.Context, chatID int64, text string)
}

func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	healthRepo *repository.HealthRepository,
	telegramRepo *repository.TelegramRepository,
	aiClient *ai.Client,
) *NotificationService {
	return &NotificationService{
		notifRepo:    notifRepo,
		healthRepo:   healthRepo,
		telegramRepo: telegramRepo,
		aiClient:     aiClient,
	}
}

// SetSendFunc sets the function used to send Telegram messages.
// Called from main.go after the bot is created to avoid circular deps.
func (s *NotificationService) SetSendFunc(fn func(ctx context.Context, chatID int64, text string)) {
	s.sendMsg = fn
}

// GetPreferences returns notification preferences for a user.
func (s *NotificationService) GetPreferences(ctx context.Context, userID string) (*model.NotificationPreferences, error) {
	return s.notifRepo.GetByUserID(ctx, userID)
}

// UpdatePreferences saves notification preferences.
func (s *NotificationService) UpdatePreferences(ctx context.Context, prefs *model.NotificationPreferences) error {
	return s.notifRepo.Upsert(ctx, prefs)
}

// OnPostSync is called after a successful data sync. It checks for new
// workouts and sleep data, sending AI-powered notifications if enabled.
func (s *NotificationService) OnPostSync(ctx context.Context, userID string) {
	if s.sendMsg == nil || s.aiClient == nil {
		return
	}

	prefs, err := s.notifRepo.GetByUserID(ctx, userID)
	if err != nil {
		slog.Warn("notification: failed to get prefs", "user_id", userID, "error", err)
		return
	}

	chatID, err := s.findChatForUser(ctx, userID)
	if err != nil {
		slog.Debug("notification: no chat for user", "user_id", userID)
		return
	}

	if prefs.WorkoutEnabled {
		s.sendWorkoutNotif(ctx, userID, chatID, prefs)
	}
	if prefs.SleepEnabled {
		s.sendSleepNotif(ctx, userID, chatID, prefs)
	}
}

// CheckScheduled runs hourly and sends any due daily/weekly summaries.
func (s *NotificationService) CheckScheduled(ctx context.Context) {
	if s.sendMsg == nil || s.aiClient == nil {
		return
	}

	now := time.Now()
	hour := now.Hour()
	weekday := int(now.Weekday())

	// Daily summaries
	dailyDue, err := s.notifRepo.ListDailyDue(ctx, hour)
	if err != nil {
		slog.Warn("notification: listing daily due", "error", err)
	}
	for _, prefs := range dailyDue {
		chatID, err := s.findChatForUser(ctx, prefs.UserID)
		if err != nil {
			continue
		}
		s.sendDailySummary(ctx, prefs.UserID, chatID)
	}

	// Weekly summaries
	weeklyDue, err := s.notifRepo.ListWeeklyDue(ctx, weekday, hour)
	if err != nil {
		slog.Warn("notification: listing weekly due", "error", err)
	}
	for _, prefs := range weeklyDue {
		chatID, err := s.findChatForUser(ctx, prefs.UserID)
		if err != nil {
			continue
		}
		s.sendWeeklySummary(ctx, prefs.UserID, chatID)
	}
}

func (s *NotificationService) sendWorkoutNotif(ctx context.Context, userID string, chatID int64, prefs *model.NotificationPreferences) {
	workout, err := s.healthRepo.GetLatestWorkout(ctx, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Warn("notification: get latest workout", "user_id", userID, "error", err)
		}
		return
	}

	// Skip if already notified about this workout
	if prefs.LastWorkoutNotifiedAt != nil && !workout.StartedAt.After(*prefs.LastWorkoutNotifiedAt) {
		return
	}

	// Build workout description
	workoutDesc := formatWorkoutForAI(workout)
	healthCtx := ai.BuildHealthContext(ctx, s.healthRepo, userID)
	prompt := fmt.Sprintf(ai.WorkoutEvaluationPrompt, workoutDesc, healthCtx)

	response, _, err := s.aiClient.Chat(ctx, ai.ChatRequest{
		SystemPrompt: prompt,
		UserMessage:  "Evaluate this workout and give me recommendations.",
	})
	if err != nil {
		slog.Warn("notification: AI workout eval", "user_id", userID, "error", err)
		return
	}

	s.sendMsg(ctx, chatID, fmt.Sprintf("🏋️ <b>Workout Analysis</b>\n\n%s", response))
	_ = s.notifRepo.UpdateLastNotified(ctx, userID, "workout")
}

func (s *NotificationService) sendSleepNotif(ctx context.Context, userID string, chatID int64, prefs *model.NotificationPreferences) {
	sleep, err := s.healthRepo.GetLatestSleep(ctx, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Warn("notification: get latest sleep", "user_id", userID, "error", err)
		}
		return
	}

	// Skip if already notified about this sleep record
	if prefs.LastSleepNotifiedAt != nil && !sleep.Date.After(*prefs.LastSleepNotifiedAt) {
		return
	}

	sleepDesc := formatSleepForAI(sleep)
	healthCtx := ai.BuildHealthContext(ctx, s.healthRepo, userID)
	prompt := fmt.Sprintf(ai.SleepEvaluationPrompt, sleepDesc, healthCtx)

	response, _, err := s.aiClient.Chat(ctx, ai.ChatRequest{
		SystemPrompt: prompt,
		UserMessage:  "Analyze my sleep and give me a morning briefing.",
	})
	if err != nil {
		slog.Warn("notification: AI sleep eval", "user_id", userID, "error", err)
		return
	}

	s.sendMsg(ctx, chatID, fmt.Sprintf("😴 <b>Sleep Analysis</b>\n\n%s", response))
	_ = s.notifRepo.UpdateLastNotified(ctx, userID, "sleep")
}

func (s *NotificationService) sendDailySummary(ctx context.Context, userID string, chatID int64) {
	healthCtx := ai.BuildHealthContext(ctx, s.healthRepo, userID)
	prompt := fmt.Sprintf(ai.DailySummaryPrompt, healthCtx)

	response, _, err := s.aiClient.Chat(ctx, ai.ChatRequest{
		SystemPrompt: prompt,
		UserMessage:  "Give me my daily health summary.",
	})
	if err != nil {
		slog.Warn("notification: AI daily summary", "user_id", userID, "error", err)
		return
	}

	s.sendMsg(ctx, chatID, fmt.Sprintf("📊 <b>Daily Summary</b>\n\n%s", response))
	_ = s.notifRepo.UpdateLastNotified(ctx, userID, "daily")
}

func (s *NotificationService) sendWeeklySummary(ctx context.Context, userID string, chatID int64) {
	healthCtx := ai.BuildHealthContext(ctx, s.healthRepo, userID)
	prompt := fmt.Sprintf(ai.WeeklySummaryPrompt, healthCtx)

	response, _, err := s.aiClient.Chat(ctx, ai.ChatRequest{
		SystemPrompt: prompt,
		UserMessage:  "Give me my weekly health review.",
	})
	if err != nil {
		slog.Warn("notification: AI weekly summary", "user_id", userID, "error", err)
		return
	}

	s.sendMsg(ctx, chatID, fmt.Sprintf("📈 <b>Weekly Review</b>\n\n%s", response))
	_ = s.notifRepo.UpdateLastNotified(ctx, userID, "weekly")
}

// findChatForUser finds an approved, non-blocked Telegram chat linked to the user.
func (s *NotificationService) findChatForUser(ctx context.Context, userID string) (int64, error) {
	chats, err := s.telegramRepo.ListAll(ctx)
	if err != nil {
		return 0, err
	}
	for _, chat := range chats {
		if chat.IsApproved && !chat.IsBlocked && chat.UserID != nil && *chat.UserID == userID {
			return chat.ChatID, nil
		}
	}
	return 0, fmt.Errorf("no chat found for user %s", userID)
}

func formatWorkoutForAI(w *model.HealthWorkout) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Type: %s", strings.ReplaceAll(w.WorkoutType, "_", " ")))
	parts = append(parts, fmt.Sprintf("Date: %s", w.StartedAt.Format("Jan 2, 15:04")))
	if w.DurationSec != nil {
		parts = append(parts, fmt.Sprintf("Duration: %d minutes", *w.DurationSec/60))
	}
	if w.DistanceM != nil && *w.DistanceM > 0 {
		parts = append(parts, fmt.Sprintf("Distance: %.2f km", float64(*w.DistanceM)/1000))
	}
	if w.Calories != nil {
		parts = append(parts, fmt.Sprintf("Calories: %d kcal", *w.Calories))
	}
	if w.AvgHeartRate != nil {
		parts = append(parts, fmt.Sprintf("Avg HR: %d bpm", *w.AvgHeartRate))
	}
	if w.MaxHeartRate != nil {
		parts = append(parts, fmt.Sprintf("Max HR: %d bpm", *w.MaxHeartRate))
	}
	if w.AvgPace != nil && *w.AvgPace > 0 {
		parts = append(parts, fmt.Sprintf("Avg Pace: %.1f min/km", *w.AvgPace))
	}
	return strings.Join(parts, "\n")
}

func formatSleepForAI(s *model.HealthSleep) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Date: %s", s.Date.Format("Jan 2")))
	if s.DurationMin != nil {
		parts = append(parts, fmt.Sprintf("Total Duration: %dh %dm", *s.DurationMin/60, *s.DurationMin%60))
	}
	if s.SleepStart != nil {
		parts = append(parts, fmt.Sprintf("Bedtime: %s", s.SleepStart.Format("15:04")))
	}
	if s.SleepEnd != nil {
		parts = append(parts, fmt.Sprintf("Wake time: %s", s.SleepEnd.Format("15:04")))
	}
	if s.DeepMin != nil {
		parts = append(parts, fmt.Sprintf("Deep sleep: %dm", *s.DeepMin))
	}
	if s.LightMin != nil {
		parts = append(parts, fmt.Sprintf("Light sleep: %dm", *s.LightMin))
	}
	if s.REMMin != nil {
		parts = append(parts, fmt.Sprintf("REM sleep: %dm", *s.REMMin))
	}
	if s.AwakeMin != nil {
		parts = append(parts, fmt.Sprintf("Awake: %dm", *s.AwakeMin))
	}
	return strings.Join(parts, "\n")
}
