package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mike/fitassist/internal/ai"
	"github.com/mike/fitassist/internal/config"
	cronpkg "github.com/mike/fitassist/internal/cron"
	"github.com/mike/fitassist/internal/database"
	"github.com/mike/fitassist/internal/repository"
	"github.com/mike/fitassist/internal/server"
	"github.com/mike/fitassist/internal/service"
	"github.com/mike/fitassist/internal/telegram"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Server.Mode == "development" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	slog.Info("starting FitAssist", "version", "0.1.0", "mode", cfg.Server.Mode)

	// Validate required config
	if cfg.Security.JWTSecret == "" {
		slog.Error("security.jwt_secret is required — generate with: openssl rand -base64 32")
		os.Exit(1)
	}
	if cfg.Security.EncryptionKey == "" {
		slog.Error("security.encryption_key is required — generate with: openssl rand -hex 32")
		os.Exit(1)
	}
	if len(cfg.Security.EncryptionKey) != 64 {
		slog.Error("security.encryption_key must be 64 hex characters (32 bytes)")
		os.Exit(1)
	}

	db, err := database.Connect(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if cfg.Database.AutoMigrate {
		slog.Info("running database migrations")
		if err := database.Migrate(db, cfg.Database); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	srv := server.New(cfg, db)

	// AI client (shared between HTTP handlers and Telegram bot)
	var aiClient *ai.Client
	if cfg.Claude.APIKey != "" {
		aiClient = ai.NewClient(cfg.Claude)
		slog.Info("AI assistant enabled", "model", cfg.Claude.Model)
	}

	// Notification service
	notifRepo := repository.NewNotificationRepository(srv.DB())
	notifSvc := service.NewNotificationService(notifRepo, srv.HealthRepo, srv.TelegramRepo, aiClient)

	// Set post-sync hook to trigger notifications
	srv.SyncService().SetPostSyncHook(notifSvc.OnPostSync)

	// Start cron scheduler for Mi Fitness sync
	var scheduler *cronpkg.Scheduler
	if cfg.MiFit.SyncIntervalMinutes > 0 {
		scheduler = cronpkg.NewScheduler(srv.SyncService(), notifSvc)
		if err := scheduler.Start(cfg.MiFit.SyncIntervalMinutes); err != nil {
			slog.Error("failed to start cron scheduler", "error", err)
		}
	}

	// Start Telegram bot
	if cfg.Telegram.Enabled && cfg.Telegram.BotToken != "" {
		tgBot := telegram.New(
			cfg.Telegram,
			srv.TelegramRepo,
			srv.UserRepo,
			srv.HealthRepo,
			srv.MiFitRepo,
			srv.MiFitService(),
			srv.SyncService(),
			notifSvc,
			aiClient,
			cfg.Security.EncryptionKey,
		)
		notifSvc.SetSendFunc(tgBot.SendToChat)
		go func() {
			if err := tgBot.Start(ctx); err != nil {
				slog.Error("telegram bot error", "error", err)
			}
		}()
	}

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		slog.Info("HTTP server listening", "addr", addr)
		if err := srv.Start(addr); err != nil {
			slog.Error("server error", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	if scheduler != nil {
		scheduler.Stop()
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("goodbye")
}
