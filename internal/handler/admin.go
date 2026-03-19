package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mike/fitassist/internal/repository"
)

type AdminHandler struct {
	userRepo     *repository.UserRepository
	telegramRepo *repository.TelegramRepository
	syncLogRepo  *repository.SyncLogRepository
	mifitRepo    *repository.MiFitRepository
	dbDSN        string
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	telegramRepo *repository.TelegramRepository,
	syncLogRepo *repository.SyncLogRepository,
	mifitRepo *repository.MiFitRepository,
	dbDSN string,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		telegramRepo: telegramRepo,
		syncLogRepo:  syncLogRepo,
		mifitRepo:    mifitRepo,
		dbDSN:        dbDSN,
	}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	users, err := h.userRepo.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	var req struct {
		Role     *string `json:"role"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AdminHandler) ListChats(w http.ResponseWriter, r *http.Request) {
	chats, err := h.telegramRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chats")
		return
	}

	writeJSON(w, http.StatusOK, chats)
}

func (h *AdminHandler) UpdateChat(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	var req struct {
		IsApproved *bool   `json:"is_approved"`
		IsBlocked  *bool   `json:"is_blocked"`
		UserID     *string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.IsApproved != nil {
		if *req.IsApproved {
			// Approve: if no user_id provided, use the requesting admin's ID
			userID := ""
			if req.UserID != nil {
				userID = *req.UserID
			} else {
				userID = GetUserID(r)
			}
			if err := h.telegramRepo.Approve(r.Context(), id, userID); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to approve chat")
				return
			}
		} else {
			// Unapprove
			if err := h.telegramRepo.Unapprove(r.Context(), id); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to unapprove chat")
				return
			}
		}
	}

	if req.IsBlocked != nil {
		if *req.IsBlocked {
			if err := h.telegramRepo.Block(r.Context(), id); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to block chat")
				return
			}
		} else {
			if err := h.telegramRepo.Unblock(r.Context(), id); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to unblock chat")
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminHandler) SyncLogs(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	logs, err := h.syncLogRepo.ListAll(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sync logs")
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

func (h *AdminHandler) Export(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "sql"
	}

	if format != "sql" && format != "json" {
		writeError(w, http.StatusBadRequest, "format must be sql or json")
		return
	}

	// For SQL export we use pg_dump via the system
	// This requires pg_dump to be available in the container
	if format == "sql" {
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", "attachment; filename=fitassist_export.sql")

		cmd := exec.CommandContext(r.Context(), "pg_dump", "--no-owner", "--no-acl", h.dbDSN)
		cmd.Stdout = w
		if err := cmd.Run(); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("export failed: %v", err))
		}
		return
	}

	// JSON export: dump each table
	writeError(w, http.StatusNotImplemented, "JSON export not implemented yet")
}

func (h *AdminHandler) Import(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		writeError(w, http.StatusBadRequest, "multipart form required")
		return
	}

	writeError(w, http.StatusNotImplemented, "import not implemented yet")
}
