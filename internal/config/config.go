package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	Claude   ClaudeConfig   `mapstructure:"claude"`
	MiFit    MiFitConfig    `mapstructure:"mifit"`
	Security SecurityConfig `mapstructure:"security"`
	Admin    AdminConfig    `mapstructure:"admin"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // development, production
}

type DatabaseConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Name        string `mapstructure:"name"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	SSLMode     string `mapstructure:"ssl_mode"`
	AutoMigrate bool   `mapstructure:"auto_migrate"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type TelegramConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	BotToken     string  `mapstructure:"bot_token"`
	AdminChatIDs []int64 `mapstructure:"admin_chat_ids"`
}

type ClaudeConfig struct {
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

type MiFitConfig struct {
	SyncIntervalMinutes int    `mapstructure:"sync_interval_minutes"`
	APIBaseURL          string `mapstructure:"api_base_url"`
}

type SecurityConfig struct {
	JWTSecret     string   `mapstructure:"jwt_secret"`
	EncryptionKey string   `mapstructure:"encryption_key"`
	CORSOrigins   []string `mapstructure:"cors_origins"`
}

type AdminConfig struct {
	InitialUsername string `mapstructure:"initial_username"`
	InitialPassword string `mapstructure:"initial_password"`
}

func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "development")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "fitassist")
	v.SetDefault("database.user", "fitassist")
	v.SetDefault("database.password", "")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.auto_migrate", true)

	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)

	v.SetDefault("telegram.enabled", false)
	v.SetDefault("telegram.bot_token", "")

	v.SetDefault("claude.api_key", "")
	v.SetDefault("claude.model", "claude-sonnet-4-5-20250929")
	v.SetDefault("claude.max_tokens", 4096)

	v.SetDefault("mifit.sync_interval_minutes", 30)
	v.SetDefault("mifit.api_base_url", "https://api-mifit-de2.huami.com")

	v.SetDefault("security.jwt_secret", "")
	v.SetDefault("security.encryption_key", "")
	v.SetDefault("security.cors_origins", []string{"http://localhost:5173"})

	v.SetDefault("admin.initial_username", "admin")
	v.SetDefault("admin.initial_password", "")

	// Config file
	v.SetConfigName("config")
	v.SetConfigType("json")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	// .env file
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.MergeInConfig()

	// Environment variables (highest priority)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Map specific env vars to nested config keys
	envBindings := map[string]string{
		"SERVER_HOST":           "server.host",
		"SERVER_PORT":           "server.port",
		"SERVER_MODE":           "server.mode",
		"DB_HOST":               "database.host",
		"DB_PORT":               "database.port",
		"DB_NAME":               "database.name",
		"DB_USER":               "database.user",
		"DB_PASSWORD":           "database.password",
		"DB_SSL_MODE":           "database.ssl_mode",
		"DB_AUTO_MIGRATE":       "database.auto_migrate",
		"REDIS_ENABLED":         "redis.enabled",
		"REDIS_HOST":            "redis.host",
		"REDIS_PORT":            "redis.port",
		"TELEGRAM_ENABLED":      "telegram.enabled",
		"TELEGRAM_BOT_TOKEN":    "telegram.bot_token",
		"CLAUDE_API_KEY":        "claude.api_key",
		"CLAUDE_MODEL":          "claude.model",
		"CLAUDE_MAX_TOKENS":     "claude.max_tokens",
		"MIFIT_SYNC_INTERVAL":   "mifit.sync_interval_minutes",
		"MIFIT_API_BASE_URL":    "mifit.api_base_url",
		"JWT_SECRET":            "security.jwt_secret",
		"ENCRYPTION_KEY":        "security.encryption_key",
		"ADMIN_USERNAME":        "admin.initial_username",
		"ADMIN_PASSWORD":        "admin.initial_password",
	}
	for env, key := range envBindings {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("binding env %s: %w", env, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
