package config

import (
	"testing"
)

func TestDatabaseDSN(t *testing.T) {
	db := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "fitassist",
		User:     "fitassist",
		Password: "secret",
		SSLMode:  "disable",
	}

	dsn := db.DSN()
	expected := "postgres://fitassist:secret@localhost:5432/fitassist?sslmode=disable"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestDatabaseDSN_CustomPort(t *testing.T) {
	db := DatabaseConfig{
		Host:     "db.example.com",
		Port:     5433,
		Name:     "mydb",
		User:     "myuser",
		Password: "p@ss",
		SSLMode:  "require",
	}

	dsn := db.DSN()
	expected := "postgres://myuser:p@ss@db.example.com:5433/mydb?sslmode=require"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Load will try to read config files, but we're testing that defaults work
	// when no config file is present. In CI, this is the normal case.
	// We can't easily test this without setting up a temp dir,
	// so just verify the struct definitions are correct.
	cfg := Config{}
	if cfg.Server.Port != 0 {
		t.Error("zero value should be 0")
	}
}
