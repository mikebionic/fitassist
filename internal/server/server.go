package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/config"
	"github.com/mike/fitassist/internal/handler"
	"github.com/mike/fitassist/internal/repository"
	"github.com/mike/fitassist/internal/service"
)

type Server struct {
	http         *http.Server
	router       chi.Router
	cfg          *config.Config
	db           *sqlx.DB
	syncService  *service.SyncService
	mifitService *service.MiFitService

	// Exported repos for telegram bot
	UserRepo     *repository.UserRepository
	HealthRepo   *repository.HealthRepository
	TelegramRepo *repository.TelegramRepository
	MiFitRepo    *repository.MiFitRepository
}

// SyncService returns the sync service for use by cron scheduler.
func (s *Server) SyncService() *service.SyncService {
	return s.syncService
}

// MiFitService returns the MiFit service for use by telegram bot.
func (s *Server) MiFitService() *service.MiFitService {
	return s.mifitService
}

func New(cfg *config.Config, db *sqlx.DB) *Server {
	r := chi.NewRouter()

	s := &Server{
		http: &http.Server{
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		router: r,
		cfg:    cfg,
		db:     db,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) Start(addr string) error {
	s.http.Addr = addr
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.http.Shutdown(ctx)
}

func (s *Server) setupRoutes() {
	// Repositories
	userRepo := repository.NewUserRepository(s.db)
	healthRepo := repository.NewHealthRepository(s.db)
	telegramRepo := repository.NewTelegramRepository(s.db)
	aiRepo := repository.NewAISessionRepository(s.db)
	mifitRepo := repository.NewMiFitRepository(s.db)
	syncLogRepo := repository.NewSyncLogRepository(s.db)

	// Services
	authService := service.NewAuthService(userRepo, s.cfg.Security)
	healthService := service.NewHealthService(healthRepo)
	syncService := service.NewSyncService(healthRepo, mifitRepo, syncLogRepo, s.cfg.MiFit.APIBaseURL, s.cfg.Security.EncryptionKey)
	mifitService := service.NewMiFitService(mifitRepo, syncService, s.cfg.MiFit.APIBaseURL, s.cfg.Security.EncryptionKey)

	// Store services and repos for external access (telegram bot, cron)
	s.syncService = syncService
	s.mifitService = mifitService
	s.UserRepo = userRepo
	s.HealthRepo = healthRepo
	s.TelegramRepo = telegramRepo
	s.MiFitRepo = mifitRepo

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(healthService)
	mifitHandler := handler.NewMiFitHandler(mifitService)
	adminHandler := handler.NewAdminHandler(userRepo, telegramRepo, syncLogRepo, mifitRepo)

	// Create initial admin
	go service.EnsureAdmin(context.Background(), userRepo, s.cfg.Admin)

	r := s.router

	// Serve frontend static files
	fileServer := http.FileServer(http.Dir("web/dist"))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(handler.AuthMiddleware(s.cfg.Security.JWTSecret))

			r.Post("/auth/logout", authHandler.Logout)

			// Health data
			r.Get("/health/dashboard", healthHandler.Dashboard)
			r.Get("/health/steps", healthHandler.Steps)
			r.Get("/health/sleep", healthHandler.Sleep)
			r.Get("/health/heartrate", healthHandler.HeartRate)
			r.Get("/health/spo2", healthHandler.SpO2)
			r.Get("/health/workouts", healthHandler.Workouts)
			r.Get("/health/workouts/{id}", healthHandler.WorkoutByID)
			r.Get("/health/stress", healthHandler.Stress)

			// AI
			r.Get("/ai/sessions", handler.Placeholder)
			r.Post("/ai/sessions", handler.Placeholder)
			r.Post("/ai/summary", handler.Placeholder)

			// Mi Fitness
			r.Post("/mifit/link", mifitHandler.Link)
			r.Post("/mifit/sync", mifitHandler.Sync)
			r.Get("/mifit/status", mifitHandler.Status)

			// User settings
			r.Get("/settings/profile", handler.Placeholder)
			r.Put("/settings/profile", handler.Placeholder)
			r.Put("/settings/notifications", handler.Placeholder)
		})

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(handler.AuthMiddleware(s.cfg.Security.JWTSecret))
			r.Use(handler.AdminMiddleware)

			r.Get("/admin/users", adminHandler.ListUsers)
			r.Patch("/admin/users/{id}", adminHandler.UpdateUser)
			r.Get("/admin/chats", adminHandler.ListChats)
			r.Patch("/admin/chats/{id}", adminHandler.UpdateChat)
			r.Get("/admin/sync-logs", adminHandler.SyncLogs)
			r.Get("/admin/export", adminHandler.Export)
			r.Post("/admin/import", adminHandler.Import)

			_ = aiRepo // will be used in phase 5
		})
	})

	// Serve frontend for all non-API routes
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
}
