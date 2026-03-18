package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/mike/fitassist/internal/ai"
	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/repository"
)

type AIHandler struct {
	claude     *ai.Client
	aiRepo     *repository.AISessionRepository
	healthRepo *repository.HealthRepository
}

func NewAIHandler(claude *ai.Client, aiRepo *repository.AISessionRepository, healthRepo *repository.HealthRepository) *AIHandler {
	return &AIHandler{
		claude:     claude,
		aiRepo:     aiRepo,
		healthRepo: healthRepo,
	}
}

// --- REST endpoints ---

// ListSessions returns all AI sessions for the current user.
func (h *AIHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	sessions, err := h.aiRepo.ListSessions(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}
	if sessions == nil {
		sessions = []model.AISession{}
	}
	writeJSON(w, http.StatusOK, sessions)
}

// CreateSession creates a new AI session.
func (h *AIHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Title = "New Chat"
	}
	if req.Title == "" {
		req.Title = "New Chat"
	}

	// Build health context for system prompt
	healthCtx := ai.BuildHealthContext(r.Context(), h.healthRepo, userID)
	systemPrompt := fmt.Sprintf(ai.SystemPromptTemplate, healthCtx)

	session := &model.AISession{
		UserID:       userID,
		Title:        &req.Title,
		SystemPrompt: &systemPrompt,
	}
	if err := h.aiRepo.CreateSession(r.Context(), session); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// GetSession returns a session with its messages.
func (h *AIHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := GetUserID(r)

	session, err := h.aiRepo.GetSession(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	messages, _ := h.aiRepo.GetMessages(r.Context(), sessionID)
	if messages == nil {
		messages = []model.AIMessage{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session":  session,
		"messages": messages,
	})
}

// DeleteSession deletes a session and its messages.
func (h *AIHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := GetUserID(r)

	session, err := h.aiRepo.GetSession(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.aiRepo.DeleteSession(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SendMessage sends a message to Claude and returns the response (non-streaming).
func (h *AIHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if h.claude == nil {
		writeError(w, http.StatusServiceUnavailable, "AI assistant is not configured")
		return
	}
	sessionID := chi.URLParam(r, "id")
	userID := GetUserID(r)

	session, err := h.aiRepo.GetSession(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// Save user message
	userMsg := &model.AIMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message,
	}
	if err := h.aiRepo.AddMessage(r.Context(), userMsg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}

	// Get history
	history, _ := h.aiRepo.GetMessages(r.Context(), sessionID)

	// Refresh health context in system prompt
	healthCtx := ai.BuildHealthContext(r.Context(), h.healthRepo, userID)
	systemPrompt := fmt.Sprintf(ai.SystemPromptTemplate, healthCtx)

	// Call Claude
	response, tokens, err := h.claude.Chat(r.Context(), ai.ChatRequest{
		SystemPrompt: systemPrompt,
		Messages:     history[:len(history)-1], // exclude the message we just added (it's the userMsg)
		UserMessage:  req.Message,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("AI error: %s", err.Error()))
		return
	}

	// Save assistant response
	assistantMsg := &model.AIMessage{
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    response,
		TokensUsed: &tokens,
	}
	_ = h.aiRepo.AddMessage(r.Context(), assistantMsg)

	writeJSON(w, http.StatusOK, assistantMsg)
}

// Summary generates a quick AI health summary (no session needed).
func (h *AIHandler) Summary(w http.ResponseWriter, r *http.Request) {
	if h.claude == nil {
		writeError(w, http.StatusServiceUnavailable, "AI assistant is not configured")
		return
	}
	userID := GetUserID(r)

	healthCtx := ai.BuildHealthContext(r.Context(), h.healthRepo, userID)
	if healthCtx == "No health data available yet. The user hasn't synced their Mi Fitness data." {
		writeJSON(w, http.StatusOK, map[string]string{
			"summary": "No health data available yet. Please sync your Mi Fitness account first.",
		})
		return
	}

	systemPrompt := fmt.Sprintf(ai.SystemPromptTemplate, healthCtx)
	response, _, err := h.claude.Chat(r.Context(), ai.ChatRequest{
		SystemPrompt: systemPrompt,
		UserMessage:  "Give me a brief health summary and 2-3 actionable recommendations based on my recent data. Keep it concise.",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("AI error: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"summary": response})
}

// --- WebSocket streaming ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsMessage is the JSON envelope for WebSocket communication.
type wsMessage struct {
	Type      string `json:"type"`                 // "message", "chunk", "done", "error", "history"
	Content   string `json:"content,omitempty"`     // text content
	SessionID string `json:"session_id,omitempty"`  // which session
	MessageID int64  `json:"message_id,omitempty"`  // saved message ID
	Tokens    int    `json:"tokens,omitempty"`       // tokens used
}

// WebSocketChat handles the WebSocket connection for streaming AI chat.
func (h *AIHandler) WebSocketChat(w http.ResponseWriter, r *http.Request) {
	if h.claude == nil {
		http.Error(w, "AI assistant is not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract auth from query param (WebSocket can't send custom headers)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	// Validate JWT manually
	userID, err := validateWSToken(r, tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	sendJSON := func(msg wsMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(msg)
	}

	slog.Info("websocket connected", "user_id", userID)

	for {
		var incoming wsMessage
		if err := conn.ReadJSON(&incoming); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			slog.Debug("websocket read error", "error", err)
			break
		}

		switch incoming.Type {
		case "message":
			h.handleWSMessage(r.Context(), userID, incoming, sendJSON)
		default:
			_ = sendJSON(wsMessage{Type: "error", Content: "unknown message type"})
		}
	}
}

func (h *AIHandler) handleWSMessage(ctx context.Context, userID string, incoming wsMessage, sendJSON func(wsMessage) error) {
	sessionID := incoming.SessionID
	if sessionID == "" {
		_ = sendJSON(wsMessage{Type: "error", Content: "session_id required"})
		return
	}

	session, err := h.aiRepo.GetSession(ctx, sessionID)
	if err != nil {
		_ = sendJSON(wsMessage{Type: "error", Content: "session not found"})
		return
	}
	if session.UserID != userID {
		_ = sendJSON(wsMessage{Type: "error", Content: "access denied"})
		return
	}

	if incoming.Content == "" {
		_ = sendJSON(wsMessage{Type: "error", Content: "message content required"})
		return
	}

	// Save user message
	userMsg := &model.AIMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   incoming.Content,
	}
	if err := h.aiRepo.AddMessage(ctx, userMsg); err != nil {
		_ = sendJSON(wsMessage{Type: "error", Content: "failed to save message"})
		return
	}

	// Get history
	history, _ := h.aiRepo.GetMessages(ctx, sessionID)

	// Build prompt
	healthCtx := ai.BuildHealthContext(ctx, h.healthRepo, userID)
	systemPrompt := fmt.Sprintf(ai.SystemPromptTemplate, healthCtx)

	// Stream response
	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	fullText, tokens, err := h.claude.ChatStream(streamCtx, ai.ChatRequest{
		SystemPrompt: systemPrompt,
		Messages:     history[:len(history)-1],
		UserMessage:  incoming.Content,
	}, func(chunk string) error {
		return sendJSON(wsMessage{
			Type:      "chunk",
			Content:   chunk,
			SessionID: sessionID,
		})
	})

	if err != nil {
		_ = sendJSON(wsMessage{Type: "error", Content: fmt.Sprintf("AI error: %s", err.Error())})
		return
	}

	// Save assistant message
	assistantMsg := &model.AIMessage{
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    fullText,
		TokensUsed: &tokens,
	}
	_ = h.aiRepo.AddMessage(ctx, assistantMsg)

	_ = sendJSON(wsMessage{
		Type:      "done",
		SessionID: sessionID,
		MessageID: assistantMsg.ID,
		Tokens:    tokens,
	})
}

// validateWSToken validates a JWT token string and returns the user ID.
func validateWSToken(r *http.Request, tokenStr string) (string, error) {
	// We need the JWT secret, but we can get it from the query context
	// The middleware isn't applied for WebSocket, so we parse it inline
	// using the same logic as AuthMiddleware but for a raw token string.
	//
	// The JWT secret is passed through the handler's context at setup time.
	jwtSecret, ok := r.Context().Value(ctxJWTSecret).(string)
	if !ok || jwtSecret == "" {
		return "", fmt.Errorf("jwt secret not configured")
	}

	return parseJWTUserID(tokenStr, jwtSecret)
}
