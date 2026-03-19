package model

import "time"

type NotificationPreferences struct {
	ID                    int64      `db:"id" json:"id"`
	UserID                string     `db:"user_id" json:"user_id"`
	DailyEnabled          bool       `db:"daily_enabled" json:"daily_enabled"`
	DailyHour             int16      `db:"daily_hour" json:"daily_hour"`
	WeeklyEnabled         bool       `db:"weekly_enabled" json:"weekly_enabled"`
	WeeklyDay             int16      `db:"weekly_day" json:"weekly_day"`
	WeeklyHour            int16      `db:"weekly_hour" json:"weekly_hour"`
	WorkoutEnabled        bool       `db:"workout_enabled" json:"workout_enabled"`
	SleepEnabled          bool       `db:"sleep_enabled" json:"sleep_enabled"`
	LastWorkoutNotifiedAt *time.Time `db:"last_workout_notified_at" json:"last_workout_notified_at,omitempty"`
	LastSleepNotifiedAt   *time.Time `db:"last_sleep_notified_at" json:"last_sleep_notified_at,omitempty"`
	LastDailyAt           *time.Time `db:"last_daily_at" json:"last_daily_at,omitempty"`
	LastWeeklyAt          *time.Time `db:"last_weekly_at" json:"last_weekly_at,omitempty"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at" json:"updated_at"`
}
