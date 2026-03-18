package handler

import (
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

func WriteHealthCheck(w http.ResponseWriter, db *sqlx.DB) {
	dbOK := true
	if err := db.Ping(); err != nil {
		dbOK = false
	}

	status := http.StatusOK
	statusText := "healthy"
	if !dbOK {
		status = http.StatusServiceUnavailable
		statusText = "unhealthy"
	}

	writeJSON(w, status, map[string]interface{}{
		"status":    statusText,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"database":  dbOK,
	})
}
