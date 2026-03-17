package model

import (
	"time"
)

type User struct {
	ID           string    `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	Email        *string   `db:"email" json:"email,omitempty"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type MiFitAccount struct {
	ID           string     `db:"id" json:"id"`
	UserID       string     `db:"user_id" json:"user_id"`
	MiEmail      string     `db:"mi_email" json:"mi_email"`
	MiPassword   []byte     `db:"mi_password" json:"-"`
	AuthToken    *string    `db:"auth_token" json:"-"`
	UserIDMi     *string    `db:"user_id_mi" json:"user_id_mi,omitempty"`
	TokenExpires *time.Time `db:"token_expires" json:"-"`
	LastSync     *time.Time `db:"last_sync" json:"last_sync,omitempty"`
	SyncEnabled  bool       `db:"sync_enabled" json:"sync_enabled"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

type TelegramChat struct {
	ID         int64     `db:"id" json:"id"`
	ChatID     int64     `db:"chat_id" json:"chat_id"`
	UserID     *string   `db:"user_id" json:"user_id,omitempty"`
	Username   *string   `db:"username" json:"username,omitempty"`
	FirstName  *string   `db:"first_name" json:"first_name,omitempty"`
	IsApproved bool      `db:"is_approved" json:"is_approved"`
	IsBlocked  bool      `db:"is_blocked" json:"is_blocked"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
