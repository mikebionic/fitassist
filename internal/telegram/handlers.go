package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mike/fitassist/internal/ai"
	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/service"
)

func (b *Bot) handleStart(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	chatID := msg.Chat.ID

	// Register the chat
	chat := &model.TelegramChat{
		ChatID: chatID,
	}
	if msg.From != nil {
		if msg.From.Username != "" {
			chat.Username = &msg.From.Username
		}
		if msg.From.FirstName != "" {
			chat.FirstName = &msg.From.FirstName
		}
	}

	_ = b.chatRepo.UpsertChat(ctx, chat)

	name := "there"
	if msg.From != nil && msg.From.FirstName != "" {
		name = msg.From.FirstName
	}

	// Auto-approve: find the first admin user and link this chat
	approved := false
	users, err := b.userRepo.List(ctx, 10, 0)
	if err == nil && len(users) > 0 {
		// Link to the first admin user (or first user if no admin)
		linkUserID := users[0].ID
		for _, u := range users {
			if u.Role == "admin" {
				linkUserID = u.ID
				break
			}
		}
		if err := b.chatRepo.Approve(ctx, chat.ID, linkUserID); err == nil {
			approved = true
		}
	}

	if approved {
		b.send(ctx, chatID, fmt.Sprintf(
			`👋 Hi <b>%s</b>! Welcome to <b>FitAssist</b>.

✅ Your chat has been <b>automatically approved</b>.

Use /link to connect your Mi Fitness account, or /help to see all commands.`, name))
	} else {
		b.send(ctx, chatID, fmt.Sprintf(
			`👋 Hi <b>%s</b>! Welcome to <b>FitAssist</b>.

⏳ Your chat is <b>pending approval</b>. Please try /start again in a moment.

Use /help to see available commands.`, name))
	}
}

func (b *Bot) handleHelp(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	b.send(ctx, update.Message.Chat.ID, `<b>Available Commands:</b>

/start — Register this chat
/link — Link your Mi Fitness account
/today — Today's summary
/week — Weekly summary
/sleep — Last night's sleep
/hr — Heart rate info
/workout — Last workout
/sync — Trigger data sync
/notify — Notification settings
/ai &lt;question&gt; — Ask AI about your health
/help — Show this help`)
}

func (b *Bot) handleLink(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Your chat is not approved yet. Please wait for admin approval.")
		return
	}

	b.linkMu.Lock()
	b.linkSessions[chatID] = &linkSession{step: "email"}
	b.linkMu.Unlock()
	b.send(ctx, chatID, "🔗 Let's link your Mi Fitness account.\n\nPlease send your <b>Xiaomi email</b>:")
}

func (b *Bot) handleDefault(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Check if we're in a link session
	b.linkMu.Lock()
	session, ok := b.linkSessions[chatID]
	b.linkMu.Unlock()
	if !ok {
		return
	}

	switch session.step {
	case "email":
		session.email = text
		session.step = "password"
		b.send(ctx, chatID, "Got it. Now send your <b>password</b>:\n\n<i>(The message will be processed and your password stored securely encrypted)</i>")

	case "password":
		userID := b.getChatUserID(ctx, chatID)
		if userID == "" {
			b.send(ctx, chatID, "⚠️ Chat not approved.")
			b.linkMu.Lock()
		delete(b.linkSessions, chatID)
		b.linkMu.Unlock()
			return
		}

		b.send(ctx, chatID, "⏳ Verifying credentials...")

		// Try to delete the password message for security
		_, _ = tgBot.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: update.Message.ID,
		})

		req := service.LinkRequest{
			Email:    session.email,
			Password: text,
		}
		result, err := b.mifitSvc.Link(ctx, userID, req)
		b.linkMu.Lock()
		delete(b.linkSessions, chatID)
		b.linkMu.Unlock()

		if err != nil {
			b.send(ctx, chatID, fmt.Sprintf("❌ Failed to link account: %s", err.Error()))
			return
		}

		b.send(ctx, chatID, fmt.Sprintf("✅ Account linked successfully!\n\nMi User ID: <code>%s</code>\n\nData sync will start automatically. Use /sync to trigger manually.", result.MiUserID))
	}
}

func (b *Bot) handleToday(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	summary, err := b.healthRepo.GetDashboardSummary(ctx, userID, today())
	if err != nil {
		b.send(ctx, chatID, "❌ Failed to get today's data.")
		return
	}

	steps := 0
	if summary.StepsToday != nil {
		steps = *summary.StepsToday
	}
	cals := 0
	if summary.CaloriesToday != nil {
		cals = *summary.CaloriesToday
	}
	dist := 0
	if summary.DistanceToday != nil {
		dist = *summary.DistanceToday
	}
	sleepMin := 0
	if summary.SleepLastNight != nil {
		sleepMin = *summary.SleepLastNight
	}
	hr := "—"
	if summary.LastHR != nil {
		hr = fmt.Sprintf("%d bpm", *summary.LastHR)
	}
	avgHR := "—"
	if summary.AvgHRToday != nil {
		avgHR = fmt.Sprintf("%.0f bpm", *summary.AvgHRToday)
	}

	b.send(ctx, chatID, fmt.Sprintf(
		`📊 <b>Today's Summary</b> — %s

🚶 Steps: <b>%s</b> (%s)
🔥 Calories: <b>%d</b> kcal
😴 Sleep: <b>%s</b>
❤️ Heart Rate: <b>%s</b> (avg: %s)`,
		time.Now().Format("Jan 2"),
		formatSteps(steps), formatDistance(dist),
		cals,
		formatDuration(sleepMin),
		hr, avgHR))
}

func (b *Bot) handleWeek(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	from := daysAgo(7)
	to := today().Add(24*time.Hour - time.Second)

	steps, _ := b.healthRepo.GetSteps(ctx, userID, from, to)
	sleeps, _ := b.healthRepo.GetSleep(ctx, userID, from, to)

	totalSteps := 0
	totalDist := 0
	totalCals := 0
	for _, s := range steps {
		if s.TotalSteps != nil {
			totalSteps += *s.TotalSteps
		}
		if s.DistanceM != nil {
			totalDist += *s.DistanceM
		}
		if s.Calories != nil {
			totalCals += *s.Calories
		}
	}

	totalSleepMin := 0
	for _, s := range sleeps {
		if s.DurationMin != nil {
			totalSleepMin += *s.DurationMin
		}
	}

	avgSteps := 0
	if len(steps) > 0 {
		avgSteps = totalSteps / len(steps)
	}
	avgSleep := 0
	if len(sleeps) > 0 {
		avgSleep = totalSleepMin / len(sleeps)
	}

	b.send(ctx, chatID, fmt.Sprintf(
		`📈 <b>Weekly Summary</b> (last 7 days)

🚶 Total Steps: <b>%s</b> (avg: %s/day)
📏 Total Distance: <b>%s</b>
🔥 Total Calories: <b>%d</b> kcal
😴 Avg Sleep: <b>%s</b>/night
📅 Days tracked: <b>%d</b>`,
		formatSteps(totalSteps), formatSteps(avgSteps),
		formatDistance(totalDist),
		totalCals,
		formatDuration(avgSleep),
		len(steps)))
}

func (b *Bot) handleSleep(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	from := daysAgo(1)
	to := today().Add(24*time.Hour - time.Second)

	sleeps, _ := b.healthRepo.GetSleep(ctx, userID, from, to)
	if len(sleeps) == 0 {
		b.send(ctx, chatID, "😴 No sleep data for last night.")
		return
	}

	s := sleeps[len(sleeps)-1]
	deep := 0
	if s.DeepMin != nil {
		deep = *s.DeepMin
	}
	light := 0
	if s.LightMin != nil {
		light = *s.LightMin
	}
	rem := 0
	if s.REMMin != nil {
		rem = *s.REMMin
	}
	awake := 0
	if s.AwakeMin != nil {
		awake = *s.AwakeMin
	}
	dur := 0
	if s.DurationMin != nil {
		dur = *s.DurationMin
	}

	bedtime := "—"
	if s.SleepStart != nil {
		bedtime = s.SleepStart.Format("15:04")
	}
	wakeup := "—"
	if s.SleepEnd != nil {
		wakeup = s.SleepEnd.Format("15:04")
	}

	b.send(ctx, chatID, fmt.Sprintf(
		`😴 <b>Last Night's Sleep</b>

⏱ Duration: <b>%s</b>
🌙 Bedtime: %s → Wake: %s

Stages:
  🟦 Deep: %s
  🟩 Light: %s
  🟪 REM: %s
  ⬜ Awake: %s`,
		formatDuration(dur), bedtime, wakeup,
		formatDuration(deep), formatDuration(light),
		formatDuration(rem), formatDuration(awake)))
}

func (b *Bot) handleHR(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	latest, err := b.healthRepo.GetLatestHeartRate(ctx, userID)
	if err != nil {
		b.send(ctx, chatID, "❤️ No heart rate data available.")
		return
	}

	from := today()
	to := from.Add(24*time.Hour - time.Second)
	todayHR, _ := b.healthRepo.GetHeartRate(ctx, userID, from, to)

	avgBPM := 0
	minBPM := 999
	maxBPM := 0
	for _, hr := range todayHR {
		bpm := int(hr.BPM)
		avgBPM += bpm
		if bpm < minBPM {
			minBPM = bpm
		}
		if bpm > maxBPM {
			maxBPM = bpm
		}
	}
	if len(todayHR) > 0 {
		avgBPM /= len(todayHR)
	}

	text := fmt.Sprintf("❤️ <b>Heart Rate</b>\n\nLast: <b>%d bpm</b> (%s)",
		latest.BPM, latest.MeasuredAt.Format("15:04"))

	if len(todayHR) > 0 {
		text += fmt.Sprintf("\n\nToday (%d readings):\n  Avg: <b>%d</b> bpm\n  Min: <b>%d</b> bpm\n  Max: <b>%d</b> bpm",
			len(todayHR), avgBPM, minBPM, maxBPM)
	}

	b.send(ctx, chatID, text)
}

func (b *Bot) handleWorkout(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	from := daysAgo(30)
	to := today().Add(24*time.Hour - time.Second)

	workouts, _ := b.healthRepo.GetWorkouts(ctx, userID, from, to)
	if len(workouts) == 0 {
		b.send(ctx, chatID, "🏋️ No workouts in the last 30 days.")
		return
	}

	w := workouts[0] // most recent (sorted DESC)
	dur := 0
	if w.DurationSec != nil {
		dur = *w.DurationSec / 60
	}
	dist := 0
	if w.DistanceM != nil {
		dist = *w.DistanceM
	}
	cal := 0
	if w.Calories != nil {
		cal = *w.Calories
	}
	avgHR := "—"
	if w.AvgHeartRate != nil {
		avgHR = fmt.Sprintf("%d", *w.AvgHeartRate)
	}
	maxHR := "—"
	if w.MaxHeartRate != nil {
		maxHR = fmt.Sprintf("%d", *w.MaxHeartRate)
	}

	typeName := strings.ReplaceAll(w.WorkoutType, "_", " ")
	typeName = strings.Title(typeName)

	b.send(ctx, chatID, fmt.Sprintf(
		`🏋️ <b>Last Workout</b>

🏃 Type: <b>%s</b>
📅 Date: %s
⏱ Duration: <b>%s</b>
📏 Distance: %s
🔥 Calories: %d kcal
❤️ HR: avg %s / max %s bpm`,
		typeName,
		w.StartedAt.Format("Jan 2, 15:04"),
		formatDuration(dur),
		formatDistance(dist),
		cal, avgHR, maxHR))
}

var dayNames = map[string]int{
	"sun": 0, "sunday": 0,
	"mon": 1, "monday": 1,
	"tue": 2, "tuesday": 2,
	"wed": 3, "wednesday": 3,
	"thu": 4, "thursday": 4,
	"fri": 5, "friday": 5,
	"sat": 6, "saturday": 6,
}

var dayLabels = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func (b *Bot) handleNotify(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	if b.notifSvc == nil {
		b.send(ctx, chatID, "⚠️ Notifications are not configured.")
		return
	}

	args := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/notify"))

	// No args — show current settings
	if args == "" {
		prefs, err := b.notifSvc.GetPreferences(ctx, userID)
		if err != nil {
			b.send(ctx, chatID, "❌ Failed to load notification settings.")
			return
		}
		b.send(ctx, chatID, formatNotifySettings(prefs))
		return
	}

	parts := strings.Fields(args)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "daily":
		b.handleNotifyDaily(ctx, chatID, userID, parts[1:])
	case "weekly":
		b.handleNotifyWeekly(ctx, chatID, userID, parts[1:])
	case "workout":
		b.handleNotifyToggle(ctx, chatID, userID, "workout", parts[1:])
	case "sleep":
		b.handleNotifyToggle(ctx, chatID, userID, "sleep", parts[1:])
	default:
		b.send(ctx, chatID, `<b>Usage:</b>
/notify — Show current settings
/notify daily HH:MM — Enable daily summary at given hour
/notify daily off — Disable daily summary
/notify weekly Mon HH:MM — Enable weekly on given day
/notify weekly off — Disable weekly summary
/notify workout on|off — Toggle workout notifications
/notify sleep on|off — Toggle sleep notifications`)
	}
}

func (b *Bot) handleNotifyDaily(ctx context.Context, chatID int64, userID string, args []string) {
	if len(args) == 0 {
		b.send(ctx, chatID, "Usage: /notify daily HH:MM or /notify daily off")
		return
	}

	prefs, err := b.notifSvc.GetPreferences(ctx, userID)
	if err != nil {
		b.send(ctx, chatID, "❌ Failed to load settings.")
		return
	}

	if strings.ToLower(args[0]) == "off" {
		prefs.DailyEnabled = false
		if err := b.notifSvc.UpdatePreferences(ctx, prefs); err != nil {
			b.send(ctx, chatID, "❌ Failed to update settings.")
			return
		}
		b.send(ctx, chatID, "✅ Daily summary <b>disabled</b>.")
		return
	}

	hour, err := parseHour(args[0])
	if err != nil {
		b.send(ctx, chatID, "❌ Invalid time. Use HH:MM format (e.g., 09:00).")
		return
	}

	prefs.DailyEnabled = true
	prefs.DailyHour = int16(hour)
	if err := b.notifSvc.UpdatePreferences(ctx, prefs); err != nil {
		b.send(ctx, chatID, "❌ Failed to update settings.")
		return
	}
	b.send(ctx, chatID, fmt.Sprintf("✅ Daily summary <b>enabled</b> at %02d:00.", hour))
}

func (b *Bot) handleNotifyWeekly(ctx context.Context, chatID int64, userID string, args []string) {
	if len(args) == 0 {
		b.send(ctx, chatID, "Usage: /notify weekly Mon HH:MM or /notify weekly off")
		return
	}

	prefs, err := b.notifSvc.GetPreferences(ctx, userID)
	if err != nil {
		b.send(ctx, chatID, "❌ Failed to load settings.")
		return
	}

	if strings.ToLower(args[0]) == "off" {
		prefs.WeeklyEnabled = false
		if err := b.notifSvc.UpdatePreferences(ctx, prefs); err != nil {
			b.send(ctx, chatID, "❌ Failed to update settings.")
			return
		}
		b.send(ctx, chatID, "✅ Weekly summary <b>disabled</b>.")
		return
	}

	if len(args) < 2 {
		b.send(ctx, chatID, "Usage: /notify weekly Mon HH:MM")
		return
	}

	day, ok := dayNames[strings.ToLower(args[0])]
	if !ok {
		b.send(ctx, chatID, "❌ Invalid day. Use: Mon, Tue, Wed, Thu, Fri, Sat, Sun.")
		return
	}

	hour, err := parseHour(args[1])
	if err != nil {
		b.send(ctx, chatID, "❌ Invalid time. Use HH:MM format (e.g., 09:00).")
		return
	}

	prefs.WeeklyEnabled = true
	prefs.WeeklyDay = int16(day)
	prefs.WeeklyHour = int16(hour)
	if err := b.notifSvc.UpdatePreferences(ctx, prefs); err != nil {
		b.send(ctx, chatID, "❌ Failed to update settings.")
		return
	}
	b.send(ctx, chatID, fmt.Sprintf("✅ Weekly summary <b>enabled</b> on %s at %02d:00.", dayLabels[day], hour))
}

func (b *Bot) handleNotifyToggle(ctx context.Context, chatID int64, userID string, notifType string, args []string) {
	if len(args) == 0 {
		b.send(ctx, chatID, fmt.Sprintf("Usage: /notify %s on|off", notifType))
		return
	}

	prefs, err := b.notifSvc.GetPreferences(ctx, userID)
	if err != nil {
		b.send(ctx, chatID, "❌ Failed to load settings.")
		return
	}

	val := strings.ToLower(args[0])
	enabled := val == "on" || val == "true" || val == "yes"

	switch notifType {
	case "workout":
		prefs.WorkoutEnabled = enabled
	case "sleep":
		prefs.SleepEnabled = enabled
	}

	if err := b.notifSvc.UpdatePreferences(ctx, prefs); err != nil {
		b.send(ctx, chatID, "❌ Failed to update settings.")
		return
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}
	b.send(ctx, chatID, fmt.Sprintf("✅ %s notifications <b>%s</b>.", strings.Title(notifType), status))
}

func formatNotifySettings(prefs *model.NotificationPreferences) string {
	daily := "off"
	if prefs.DailyEnabled {
		daily = fmt.Sprintf("on, at %02d:00", prefs.DailyHour)
	}
	weekly := "off"
	if prefs.WeeklyEnabled {
		day := "Mon"
		if int(prefs.WeeklyDay) < len(dayLabels) {
			day = dayLabels[prefs.WeeklyDay]
		}
		weekly = fmt.Sprintf("on, %s at %02d:00", day, prefs.WeeklyHour)
	}
	workout := "off"
	if prefs.WorkoutEnabled {
		workout = "on"
	}
	sleep := "off"
	if prefs.SleepEnabled {
		sleep = "on"
	}

	return fmt.Sprintf(`🔔 <b>Notification Settings</b>

📊 Daily summary: <b>%s</b>
📈 Weekly summary: <b>%s</b>
🏋️ Workout analysis: <b>%s</b>
😴 Sleep analysis: <b>%s</b>

<b>Commands:</b>
/notify daily HH:MM — Enable daily
/notify daily off — Disable daily
/notify weekly Mon HH:MM — Enable weekly
/notify weekly off — Disable weekly
/notify workout on|off
/notify sleep on|off`, daily, weekly, workout, sleep)
}

func parseHour(s string) (int, error) {
	// Accept "HH:MM" or just "HH"
	parts := strings.Split(s, ":")
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, fmt.Errorf("invalid hour")
	}
	return hour, nil
}

func (b *Bot) handleSync(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	b.send(ctx, chatID, "⏳ Syncing data from Mi Fitness...")

	err := b.mifitSvc.TriggerSync(ctx, userID)
	if err != nil {
		b.send(ctx, chatID, fmt.Sprintf("❌ Sync failed: %s", err.Error()))
		return
	}

	b.send(ctx, chatID, "✅ Sync completed! Use /today to see your data.")
}

func (b *Bot) handleAI(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	userID := b.getChatUserID(ctx, chatID)

	if userID == "" {
		b.send(ctx, chatID, "⚠️ Chat not approved or not linked to a user.")
		return
	}

	if b.aiClient == nil {
		b.send(ctx, chatID, "⚠️ AI assistant is not configured. Set the Claude API key in config.")
		return
	}

	question := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/ai"))
	if question == "" {
		b.send(ctx, chatID, "Usage: /ai &lt;your question&gt;\n\nExample: /ai How was my sleep this week?")
		return
	}

	b.send(ctx, chatID, "🤔 Thinking...")

	healthCtx := ai.BuildHealthContext(ctx, b.healthRepo, userID)
	systemPrompt := fmt.Sprintf(ai.SystemPromptTemplate, healthCtx)

	response, _, err := b.aiClient.Chat(ctx, ai.ChatRequest{
		SystemPrompt: systemPrompt,
		UserMessage:  question,
	})
	if err != nil {
		b.send(ctx, chatID, fmt.Sprintf("❌ AI error: %s", err.Error()))
		return
	}

	b.send(ctx, chatID, response)
}
