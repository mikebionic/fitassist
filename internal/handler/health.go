package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mike/fitassist/internal/service"
)

type HealthHandler struct {
	health *service.HealthService
}

func NewHealthHandler(health *service.HealthService) *HealthHandler {
	return &HealthHandler{health: health}
}

func (h *HealthHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	summary, err := h.health.GetDashboard(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h *HealthHandler) Steps(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	from, to := parseDateRange(r)

	data, err := h.health.GetSteps(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get steps")
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *HealthHandler) Sleep(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	from, to := parseDateRange(r)

	data, err := h.health.GetSleep(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get sleep data")
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *HealthHandler) HeartRate(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	from, to := parseDateRange(r)

	data, err := h.health.GetHeartRate(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get heart rate data")
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *HealthHandler) SpO2(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	from, to := parseDateRange(r)

	data, err := h.health.GetSpO2(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get SpO2 data")
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *HealthHandler) Workouts(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	from, to := parseDateRange(r)

	data, err := h.health.GetWorkouts(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get workouts")
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *HealthHandler) WorkoutByID(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workout id")
		return
	}

	data, err := h.health.GetWorkoutByID(r.Context(), userID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workout not found")
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (h *HealthHandler) Stress(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	from, to := parseDateRange(r)

	data, err := h.health.GetStress(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stress data")
		return
	}

	writeJSON(w, http.StatusOK, data)
}
