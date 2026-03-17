package model

import "time"

type AISession struct {
	ID           string    `db:"id" json:"id"`
	UserID       string    `db:"user_id" json:"user_id"`
	Title        *string   `db:"title" json:"title,omitempty"`
	SystemPrompt *string   `db:"system_prompt" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type AIMessage struct {
	ID         int64     `db:"id" json:"id"`
	SessionID  string    `db:"session_id" json:"session_id"`
	Role       string    `db:"role" json:"role"`
	Content    string    `db:"content" json:"content"`
	TokensUsed *int      `db:"tokens_used" json:"tokens_used,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type SyncLog struct {
	ID            int64      `db:"id" json:"id"`
	UserID        string     `db:"user_id" json:"user_id"`
	SyncType      *string    `db:"sync_type" json:"sync_type,omitempty"`
	Status        *string    `db:"status" json:"status,omitempty"`
	RecordsSynced int        `db:"records_synced" json:"records_synced"`
	ErrorMessage  *string    `db:"error_message" json:"error_message,omitempty"`
	StartedAt     time.Time  `db:"started_at" json:"started_at"`
	FinishedAt    *time.Time `db:"finished_at" json:"finished_at,omitempty"`
}

type AppSetting struct {
	Key       string    `db:"key" json:"key"`
	Value     *string   `db:"value" json:"value"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
