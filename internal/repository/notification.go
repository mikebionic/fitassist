package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/model"
)

type NotificationRepository struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// GetByUserID returns notification preferences for a user, creating defaults if none exist.
func (r *NotificationRepository) GetByUserID(ctx context.Context, userID string) (*model.NotificationPreferences, error) {
	var prefs model.NotificationPreferences
	err := r.db.GetContext(ctx, &prefs,
		"SELECT * FROM notification_preferences WHERE user_id = $1", userID)
	if errors.Is(err, sql.ErrNoRows) {
		// Create defaults
		prefs = model.NotificationPreferences{
			UserID:         userID,
			WorkoutEnabled: true,
			SleepEnabled:   true,
			DailyHour:      9,
			WeeklyDay:      1,
			WeeklyHour:     9,
		}
		if err := r.Upsert(ctx, &prefs); err != nil {
			return nil, err
		}
		return &prefs, nil
	}
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

// Upsert inserts or updates notification preferences.
func (r *NotificationRepository) Upsert(ctx context.Context, prefs *model.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			user_id, daily_enabled, daily_hour, weekly_enabled, weekly_day, weekly_hour,
			workout_enabled, sleep_enabled, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (user_id) DO UPDATE SET
			daily_enabled = EXCLUDED.daily_enabled,
			daily_hour = EXCLUDED.daily_hour,
			weekly_enabled = EXCLUDED.weekly_enabled,
			weekly_day = EXCLUDED.weekly_day,
			weekly_hour = EXCLUDED.weekly_hour,
			workout_enabled = EXCLUDED.workout_enabled,
			sleep_enabled = EXCLUDED.sleep_enabled,
			updated_at = now()
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		prefs.UserID, prefs.DailyEnabled, prefs.DailyHour,
		prefs.WeeklyEnabled, prefs.WeeklyDay, prefs.WeeklyHour,
		prefs.WorkoutEnabled, prefs.SleepEnabled,
	).Scan(&prefs.ID, &prefs.CreatedAt, &prefs.UpdatedAt)
}

// UpdateLastNotified bumps the timestamp for a specific notification type.
func (r *NotificationRepository) UpdateLastNotified(ctx context.Context, userID string, notifType string) error {
	var col string
	switch notifType {
	case "workout":
		col = "last_workout_notified_at"
	case "sleep":
		col = "last_sleep_notified_at"
	case "daily":
		col = "last_daily_at"
	case "weekly":
		col = "last_weekly_at"
	default:
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE notification_preferences SET "+col+" = now(), updated_at = now() WHERE user_id = $1",
		userID)
	return err
}

// ListDailyDue returns users whose daily summary is due at the given hour.
func (r *NotificationRepository) ListDailyDue(ctx context.Context, hour int) ([]model.NotificationPreferences, error) {
	var prefs []model.NotificationPreferences
	err := r.db.SelectContext(ctx, &prefs, `
		SELECT * FROM notification_preferences
		WHERE daily_enabled = true
		  AND daily_hour = $1
		  AND (last_daily_at IS NULL OR last_daily_at < $2)`,
		hour, time.Now().Truncate(24*time.Hour))
	return prefs, err
}

// ListWeeklyDue returns users whose weekly summary is due at the given weekday and hour.
func (r *NotificationRepository) ListWeeklyDue(ctx context.Context, weekday int, hour int) ([]model.NotificationPreferences, error) {
	var prefs []model.NotificationPreferences
	err := r.db.SelectContext(ctx, &prefs, `
		SELECT * FROM notification_preferences
		WHERE weekly_enabled = true
		  AND weekly_day = $1
		  AND weekly_hour = $2
		  AND (last_weekly_at IS NULL OR last_weekly_at < $3)`,
		weekday, hour, time.Now().Truncate(24*time.Hour))
	return prefs, err
}
