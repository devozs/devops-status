package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type IncidentsHandler struct {
	store *store.Store
}

func NewIncidentsHandler(s *store.Store) *IncidentsHandler {
	return &IncidentsHandler{store: s}
}

func (h *IncidentsHandler) List(w http.ResponseWriter, r *http.Request) {
	incidents, err := h.store.ListIncidents(r.Context(), 50, "")
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}
	handler.WriteJSON(w, http.StatusOK, incidents)
}

func (h *IncidentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	inc, err := h.store.GetIncidentByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	updates, _ := h.store.ListIncidentUpdates(r.Context(), id)
	handler.WriteJSON(w, http.StatusOK, map[string]any{"incident": inc, "updates": updates})
}

type updateIncidentRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h *IncidentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateIncidentRequest
	if err := handler.DecodeJSON(r, &req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if req.Status != "" {
		if err := h.store.UpdateIncidentStatus(r.Context(), id, req.Status); err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "failed to update status")
			return
		}
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	username := "admin"
	user, err := h.store.GetAdminUserByID(r.Context(), userID)
	if err == nil {
		username = user.Username
	}

	if req.Message != "" {
		status := req.Status
		if status == "" {
			status = "update"
		}
		_ = h.store.CreateIncidentUpdate(r.Context(), id, status, req.Message, username)
	}

	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "incident", &id, req, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}
