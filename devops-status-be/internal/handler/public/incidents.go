package public

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	targetType := r.URL.Query().Get("target_type")

	incidents, err := h.store.ListIncidents(r.Context(), limit, targetType)
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

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"incident": inc,
		"updates":  updates,
	})
}
