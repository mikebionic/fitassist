package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mike/fitassist/internal/config"
	"github.com/mike/fitassist/internal/repository"
	"github.com/mike/fitassist/internal/service"
)

type Bot struct {
	bot          *bot.Bot
	cfg          config.TelegramConfig
	chatRepo     *repository.TelegramRepository
	userRepo     *repository.UserRepository
	healthRepo   *repository.HealthRepository
	mifitRepo    *repository.MiFitRepository
	mifitSvc     *service.MiFitService
	syncSvc      *service.SyncService
	encKey       string
	linkSessions map[int64]*linkSession // chat_id -> session
}

type linkSession struct {
	step  string // "email" or "password"
	email string
}

func New(
	cfg config.TelegramConfig,
	chatRepo *repository.TelegramRepository,
	userRepo *repository.UserRepository,
	healthRepo *repository.HealthRepository,
	mifitRepo *repository.MiFitRepository,
	mifitSvc *service.MiFitService,
	syncSvc *service.SyncService,
	encKey string,
) *Bot {
	return &Bot{
		cfg:          cfg,
		chatRepo:     chatRepo,
		userRepo:     userRepo,
		healthRepo:   healthRepo,
		mifitRepo:    mifitRepo,
		mifitSvc:     mifitSvc,
		syncSvc:      syncSvc,
		encKey:       encKey,
		linkSessions: make(map[int64]*linkSession),
	}
}

func (b *Bot) Start(ctx context.Context) error {
	opts := []bot.Option{
		bot.WithDefaultHandler(b.handleDefault),
	}

	tgBot, err := bot.New(b.cfg.BotToken, opts...)
	if err != nil {
		return fmt.Errorf("creating bot: %w", err)
	}
	b.bot = tgBot

	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, b.handleStart)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/link", bot.MatchTypePrefix, b.handleLink)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/today", bot.MatchTypePrefix, b.handleToday)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/week", bot.MatchTypePrefix, b.handleWeek)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/sleep", bot.MatchTypePrefix, b.handleSleep)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/hr", bot.MatchTypePrefix, b.handleHR)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/workout", bot.MatchTypePrefix, b.handleWorkout)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix, b.handleHelp)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/sync", bot.MatchTypePrefix, b.handleSync)

	slog.Info("telegram bot started")
	b.bot.Start(ctx)
	return nil
}

func (b *Bot) send(ctx context.Context, chatID int64, text string) {
	_, err := b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.Warn("failed to send telegram message", "chat_id", chatID, "error", err)
	}
}

// SendToChat sends a message to a specific chat (used by notification service).
func (b *Bot) SendToChat(ctx context.Context, chatID int64, text string) {
	b.send(ctx, chatID, text)
}

// getChatUserID returns the linked user ID for a chat, or empty if not approved.
func (b *Bot) getChatUserID(ctx context.Context, chatID int64) string {
	chat, err := b.chatRepo.GetByChatID(ctx, chatID)
	if err != nil {
		return ""
	}
	if !chat.IsApproved || chat.IsBlocked {
		return ""
	}
	if chat.UserID == nil {
		return ""
	}
	return *chat.UserID
}

func formatSteps(steps int) string {
	if steps == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", steps)
}

func formatDuration(min int) string {
	h := min / 60
	m := min % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func formatDistance(meters int) string {
	if meters == 0 {
		return "0 m"
	}
	if meters < 1000 {
		return fmt.Sprintf("%d m", meters)
	}
	return fmt.Sprintf("%.1f km", float64(meters)/1000)
}

func today() time.Time {
	return time.Now().Truncate(24 * time.Hour)
}

func daysAgo(n int) time.Time {
	return today().AddDate(0, 0, -n)
}
