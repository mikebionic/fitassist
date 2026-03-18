package handler

import (
	"encoding/json"
	"net/http"

	"github.com/mike/fitassist/internal/service"
)

type MiFitHandler struct {
	mifitSvc *service.MiFitService
}

func NewMiFitHandler(mifitSvc *service.MiFitService) *MiFitHandler {
	return &MiFitHandler{mifitSvc: mifitSvc}
}

func (h *MiFitHandler) Link(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var req service.LinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	result, err := h.mifitSvc.Link(r.Context(), userID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MiFitHandler) Sync(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	if err := h.mifitSvc.TriggerSync(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sync started"})
}

func (h *MiFitHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	status, err := h.mifitSvc.GetStatus(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, status)
}
