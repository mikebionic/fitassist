package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/model"
)

type MiFitRepository struct {
	db *sqlx.DB
}

func NewMiFitRepository(db *sqlx.DB) *MiFitRepository {
	return &MiFitRepository{db: db}
}

func (r *MiFitRepository) Create(ctx context.Context, acc *model.MiFitAccount) error {
	query := `
		INSERT INTO mifit_accounts (user_id, mi_email, mi_password, sync_enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		acc.UserID, acc.MiEmail, acc.MiPassword, acc.SyncEnabled,
	).Scan(&acc.ID, &acc.CreatedAt)
}

func (r *MiFitRepository) GetByUserID(ctx context.Context, userID string) (*model.MiFitAccount, error) {
	var acc model.MiFitAccount
	err := r.db.GetContext(ctx, &acc,
		"SELECT * FROM mifit_accounts WHERE user_id = $1", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &acc, err
}

func (r *MiFitRepository) UpdateToken(ctx context.Context, id string, token string, userIDMi string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mifit_accounts SET auth_token = $1, user_id_mi = $2 WHERE id = $3`,
		token, userIDMi, id)
	return err
}

func (r *MiFitRepository) UpdateAuthMethod(ctx context.Context, id string, authMethod string, xiaomiAuthData []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mifit_accounts SET auth_method = $1, xiaomi_auth_data = $2 WHERE id = $3`,
		authMethod, xiaomiAuthData, id)
	return err
}

func (r *MiFitRepository) UpdateLastSync(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE mifit_accounts SET last_sync = now() WHERE id = $1", id)
	return err
}

func (r *MiFitRepository) ListSyncEnabled(ctx context.Context) ([]model.MiFitAccount, error) {
	var accounts []model.MiFitAccount
	err := r.db.SelectContext(ctx, &accounts,
		"SELECT * FROM mifit_accounts WHERE sync_enabled = true")
	return accounts, err
}

func (r *MiFitRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM mifit_accounts WHERE id = $1", id)
	return err
}
