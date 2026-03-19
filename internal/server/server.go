package server

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/ai"
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

// DB returns the database connection for creating additional repositories.
func (s *Server) DB() *sqlx.DB {
	return s.db
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

	// AI client (only if API key is configured)
	var claudeClient *ai.Client
	if s.cfg.Claude.APIKey != "" {
		claudeClient = ai.NewClient(s.cfg.Claude)
	}

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(healthService)
	mifitHandler := handler.NewMiFitHandler(mifitService)
	adminHandler := handler.NewAdminHandler(userRepo, telegramRepo, syncLogRepo, mifitRepo, s.cfg.Database.DSN())
	aiHandler := handler.NewAIHandler(claudeClient, aiRepo, healthRepo)

	// Create initial admin
	service.EnsureAdmin(context.Background(), userRepo, s.cfg.Admin)

	r := s.router

	// Health check (public, no auth)
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteHealthCheck(w, s.db)
	})

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
			r.Get("/ai/sessions", aiHandler.ListSessions)
			r.Post("/ai/sessions", aiHandler.CreateSession)
			r.Get("/ai/sessions/{id}", aiHandler.GetSession)
			r.Delete("/ai/sessions/{id}", aiHandler.DeleteSession)
			r.Post("/ai/sessions/{id}/messages", aiHandler.SendMessage)
			r.Post("/ai/summary", aiHandler.Summary)

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

		})
	})

	// WebSocket route (outside /api, with JWT secret in context for token validation)
	r.Group(func(r chi.Router) {
		r.Use(handler.JWTSecretMiddleware(s.cfg.Security.JWTSecret))
		r.Get("/ws/ai/chat", aiHandler.WebSocketChat)
	})

	// Serve frontend with SPA fallback (serves index.html for unknown routes)
	spaHandler := spaFileServer("web/dist")
	r.Get("/*", spaHandler)
}

// spaFileServer serves static files from dir, falling back to index.html
// for any path that doesn't match a real file (SPA client-side routing).
func spaFileServer(dir string) http.HandlerFunc {
	fsys := http.Dir(dir)
	fileServer := http.FileServer(fsys)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if the file exists
		cleanPath := filepath.Clean(strings.TrimPrefix(path, "/"))
		fullPath := filepath.Join(dir, cleanPath)
		if _, err := os.Stat(fullPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if it's a directory with an index.html
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			if _, err := fs.Stat(os.DirFS(dir), filepath.Join(cleanPath, "index.html")); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fallback: serve index.html for client-side routing
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	}
}
