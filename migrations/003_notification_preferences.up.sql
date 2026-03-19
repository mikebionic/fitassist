CREATE TABLE IF NOT EXISTS notification_preferences (
    id                       BIGSERIAL PRIMARY KEY,
    user_id                  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    daily_enabled            BOOLEAN DEFAULT false,
    daily_hour               SMALLINT DEFAULT 9,
    weekly_enabled           BOOLEAN DEFAULT false,
    weekly_day               SMALLINT DEFAULT 1,
    weekly_hour              SMALLINT DEFAULT 9,
    workout_enabled          BOOLEAN DEFAULT true,
    sleep_enabled            BOOLEAN DEFAULT true,
    last_workout_notified_at TIMESTAMPTZ,
    last_sleep_notified_at   TIMESTAMPTZ,
    last_daily_at            TIMESTAMPTZ,
    last_weekly_at           TIMESTAMPTZ,
    created_at               TIMESTAMPTZ DEFAULT now(),
    updated_at               TIMESTAMPTZ DEFAULT now()
);
