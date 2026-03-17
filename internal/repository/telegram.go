package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/model"
)

type TelegramRepository struct {
	db *sqlx.DB
}

func NewTelegramRepository(db *sqlx.DB) *TelegramRepository {
	return &TelegramRepository{db: db}
}

func (r *TelegramRepository) UpsertChat(ctx context.Context, chat *model.TelegramChat) error {
	query := `
		INSERT INTO telegram_chats (chat_id, username, first_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name
		RETURNING id, is_approved, is_blocked, created_at`
	return r.db.QueryRowxContext(ctx, query,
		chat.ChatID, chat.Username, chat.FirstName,
	).Scan(&chat.ID, &chat.IsApproved, &chat.IsBlocked, &chat.CreatedAt)
}

func (r *TelegramRepository) GetByChatID(ctx context.Context, chatID int64) (*model.TelegramChat, error) {
	var chat model.TelegramChat
	err := r.db.GetContext(ctx, &chat, "SELECT * FROM telegram_chats WHERE chat_id = $1", chatID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &chat, err
}

func (r *TelegramRepository) ListPending(ctx context.Context) ([]model.TelegramChat, error) {
	var chats []model.TelegramChat
	err := r.db.SelectContext(ctx, &chats,
		"SELECT * FROM telegram_chats WHERE is_approved = false AND is_blocked = false ORDER BY created_at DESC")
	return chats, err
}

func (r *TelegramRepository) ListAll(ctx context.Context) ([]model.TelegramChat, error) {
	var chats []model.TelegramChat
	err := r.db.SelectContext(ctx, &chats,
		"SELECT * FROM telegram_chats ORDER BY created_at DESC")
	return chats, err
}

func (r *TelegramRepository) Approve(ctx context.Context, id int64, userID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE telegram_chats SET is_approved = true, user_id = $1 WHERE id = $2",
		userID, id)
	return err
}

func (r *TelegramRepository) Block(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE telegram_chats SET is_blocked = true WHERE id = $1", id)
	return err
}

func (r *TelegramRepository) LinkUser(ctx context.Context, chatID int64, userID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE telegram_chats SET user_id = $1 WHERE chat_id = $2",
		userID, chatID)
	return err
}
