package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["hello"] != "world" {
		t.Errorf("expected world, got %s", body["hello"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "bad input" {
		t.Errorf("expected 'bad input', got %q", body["error"])
	}
}

func TestParseDateRange_Defaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	from, to := parseDateRange(req)

	if from.IsZero() || to.IsZero() {
		t.Error("from and to should not be zero")
	}
	if to.Before(from) {
		t.Error("to should be after from")
	}
}

func TestParseDateRange_Custom(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?from=2026-01-01&to=2026-01-31", nil)
	from, to := parseDateRange(req)

	if from.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("from: got %s, want 2026-01-01", from.Format("2006-01-02"))
	}
	// to should be end of Jan 31
	if to.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("to: got %s, want 2026-01-31", to.Format("2006-01-02"))
	}
}

func TestParseIntParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?limit=25", nil)
	v := parseIntParam(req, "limit", 50)
	if v != 25 {
		t.Errorf("expected 25, got %d", v)
	}
}

func TestParseIntParam_Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	v := parseIntParam(req, "limit", 50)
	if v != 50 {
		t.Errorf("expected 50, got %d", v)
	}
}

func TestParseIntParam_Invalid(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?limit=abc", nil)
	v := parseIntParam(req, "limit", 50)
	if v != 50 {
		t.Errorf("expected 50 for invalid input, got %d", v)
	}
}

func TestPlaceholder(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	Placeholder(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", w.Code)
	}
}
