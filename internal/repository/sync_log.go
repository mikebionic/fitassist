package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/model"
)

type SyncLogRepository struct {
	db *sqlx.DB
}

func NewSyncLogRepository(db *sqlx.DB) *SyncLogRepository {
	return &SyncLogRepository{db: db}
}

func (r *SyncLogRepository) Create(ctx context.Context, log *model.SyncLog) error {
	query := `
		INSERT INTO sync_logs (user_id, sync_type, status)
		VALUES ($1, $2, $3)
		RETURNING id, started_at`
	return r.db.QueryRowxContext(ctx, query,
		log.UserID, log.SyncType, log.Status,
	).Scan(&log.ID, &log.StartedAt)
}

func (r *SyncLogRepository) Finish(ctx context.Context, id int64, status string, recordsSynced int, errMsg *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sync_logs SET status = $1, records_synced = $2, error_message = $3, finished_at = now() WHERE id = $4`,
		status, recordsSynced, errMsg, id)
	return err
}

func (r *SyncLogRepository) ListByUser(ctx context.Context, userID string, limit int) ([]model.SyncLog, error) {
	var logs []model.SyncLog
	err := r.db.SelectContext(ctx, &logs,
		"SELECT * FROM sync_logs WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2",
		userID, limit)
	return logs, err
}

func (r *SyncLogRepository) ListAll(ctx context.Context, limit, offset int) ([]model.SyncLog, error) {
	var logs []model.SyncLog
	err := r.db.SelectContext(ctx, &logs,
		"SELECT * FROM sync_logs ORDER BY started_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	return logs, err
}
