package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/model"
)

type AISessionRepository struct {
	db *sqlx.DB
}

func NewAISessionRepository(db *sqlx.DB) *AISessionRepository {
	return &AISessionRepository{db: db}
}

func (r *AISessionRepository) CreateSession(ctx context.Context, s *model.AISession) error {
	query := `
		INSERT INTO ai_sessions (user_id, title, system_prompt)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		s.UserID, s.Title, s.SystemPrompt,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *AISessionRepository) GetSession(ctx context.Context, id string) (*model.AISession, error) {
	var s model.AISession
	err := r.db.GetContext(ctx, &s, "SELECT * FROM ai_sessions WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &s, err
}

func (r *AISessionRepository) ListSessions(ctx context.Context, userID string) ([]model.AISession, error) {
	var sessions []model.AISession
	err := r.db.SelectContext(ctx, &sessions,
		"SELECT * FROM ai_sessions WHERE user_id = $1 ORDER BY updated_at DESC",
		userID)
	return sessions, err
}

func (r *AISessionRepository) UpdateSessionTitle(ctx context.Context, id, title string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE ai_sessions SET title = $1, updated_at = now() WHERE id = $2",
		title, id)
	return err
}

func (r *AISessionRepository) DeleteSession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ai_sessions WHERE id = $1", id)
	return err
}

// Messages

func (r *AISessionRepository) AddMessage(ctx context.Context, m *model.AIMessage) error {
	query := `
		INSERT INTO ai_messages (session_id, role, content, tokens_used)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := r.db.QueryRowxContext(ctx, query,
		m.SessionID, m.Role, m.Content, m.TokensUsed,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return err
	}

	// Update session timestamp
	_, _ = r.db.ExecContext(ctx,
		"UPDATE ai_sessions SET updated_at = now() WHERE id = $1", m.SessionID)

	return nil
}

func (r *AISessionRepository) GetMessages(ctx context.Context, sessionID string) ([]model.AIMessage, error) {
	var messages []model.AIMessage
	err := r.db.SelectContext(ctx, &messages,
		"SELECT * FROM ai_messages WHERE session_id = $1 ORDER BY created_at",
		sessionID)
	return messages, err
}
