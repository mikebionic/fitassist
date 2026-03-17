-- Users
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(100) NOT NULL UNIQUE,
    email           VARCHAR(255),
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- Mi Fitness accounts
CREATE TABLE IF NOT EXISTS mifit_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mi_email        TEXT NOT NULL,
    mi_password     BYTEA NOT NULL,
    auth_token      TEXT,
    user_id_mi      VARCHAR(100),
    token_expires   TIMESTAMPTZ,
    last_sync       TIMESTAMPTZ,
    sync_enabled    BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_mifit_accounts_user ON mifit_accounts(user_id);

-- Telegram chats
CREATE TABLE IF NOT EXISTS telegram_chats (
    id              BIGSERIAL PRIMARY KEY,
    chat_id         BIGINT NOT NULL UNIQUE,
    user_id         UUID REFERENCES users(id),
    username        VARCHAR(255),
    first_name      VARCHAR(255),
    is_approved     BOOLEAN DEFAULT false,
    is_blocked      BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_telegram_chats_user ON telegram_chats(user_id);

-- Steps & activity
CREATE TABLE IF NOT EXISTS health_steps (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    total_steps     INTEGER,
    distance_m      INTEGER,
    calories        INTEGER,
    active_minutes  INTEGER,
    stages          JSONB,
    UNIQUE(user_id, date)
);

-- Sleep
CREATE TABLE IF NOT EXISTS health_sleep (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    sleep_start     TIMESTAMPTZ,
    sleep_end       TIMESTAMPTZ,
    duration_min    INTEGER,
    deep_min        INTEGER,
    light_min       INTEGER,
    rem_min         INTEGER,
    awake_min       INTEGER,
    stages          JSONB,
    UNIQUE(user_id, date)
);

-- Heart rate
CREATE TABLE IF NOT EXISTS health_heartrate (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    measured_at     TIMESTAMPTZ NOT NULL,
    bpm             SMALLINT NOT NULL,
    type            VARCHAR(20) DEFAULT 'auto',
    UNIQUE(user_id, measured_at)
);

CREATE INDEX idx_heartrate_user_date ON health_heartrate(user_id, measured_at);

-- SpO2
CREATE TABLE IF NOT EXISTS health_spo2 (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    measured_at     TIMESTAMPTZ NOT NULL,
    value           SMALLINT NOT NULL,
    UNIQUE(user_id, measured_at)
);

CREATE INDEX idx_spo2_user_date ON health_spo2(user_id, measured_at);

-- Workouts
CREATE TABLE IF NOT EXISTS health_workouts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workout_type    VARCHAR(50) NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    duration_sec    INTEGER,
    distance_m      INTEGER,
    calories        INTEGER,
    avg_heartrate   SMALLINT,
    max_heartrate   SMALLINT,
    avg_pace        FLOAT,
    route_data      JSONB,
    details         JSONB,
    UNIQUE(user_id, started_at)
);

CREATE INDEX idx_workouts_user_date ON health_workouts(user_id, started_at);

-- Stress
CREATE TABLE IF NOT EXISTS health_stress (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    measured_at     TIMESTAMPTZ NOT NULL,
    value           SMALLINT NOT NULL,
    UNIQUE(user_id, measured_at)
);

CREATE INDEX idx_stress_user_date ON health_stress(user_id, measured_at);

-- AI sessions
CREATE TABLE IF NOT EXISTS ai_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(500),
    system_prompt   TEXT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_ai_sessions_user ON ai_sessions(user_id);

-- AI messages
CREATE TABLE IF NOT EXISTS ai_messages (
    id              BIGSERIAL PRIMARY KEY,
    session_id      UUID NOT NULL REFERENCES ai_sessions(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,
    content         TEXT NOT NULL,
    tokens_used     INTEGER,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_ai_messages_session ON ai_messages(session_id);

-- Sync logs
CREATE TABLE IF NOT EXISTS sync_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sync_type       VARCHAR(50),
    status          VARCHAR(20),
    records_synced  INTEGER DEFAULT 0,
    error_message   TEXT,
    started_at      TIMESTAMPTZ DEFAULT now(),
    finished_at     TIMESTAMPTZ
);

CREATE INDEX idx_sync_logs_user ON sync_logs(user_id);

-- App settings (key-value)
CREATE TABLE IF NOT EXISTS app_settings (
    key             VARCHAR(100) PRIMARY KEY,
    value           TEXT,
    updated_at      TIMESTAMPTZ DEFAULT now()
);
