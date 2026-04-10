package public

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/incidentenrich"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
)

type IncidentsHandler struct {
	store *store.Store
}

func NewIncidentsHandler(s *store.Store) *IncidentsHandler {
	return &IncidentsHandler{store: s}
}

func (h *IncidentsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	targetType := r.URL.Query().Get("target_type")

	var sincePtr, untilPtr *time.Time
	if v := r.URL.Query().Get("since"); v != "" {
		t, err := parseIncidentTimeQuery(v)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid since")
			return
		}
		sincePtr = &t
	}
	if v := r.URL.Query().Get("until"); v != "" {
		t, err := parseIncidentTimeQuery(v)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid until")
			return
		}
		untilPtr = &t
	}

	incidents, err := h.store.ListIncidents(ctx, limit, targetType, sincePtr, untilPtr)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}
	out, err := incidentenrich.List(ctx, h.store, incidents)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}
	handler.WriteJSON(w, http.StatusOK, out)
}

func parseIncidentTimeQuery(v string) (time.Time, error) {
	if t, err := time.ParseInLocation("2006-01-02", v, time.UTC); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, v)
}

func (h *IncidentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	inc, err := h.store.GetIncidentByID(ctx, id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}

	updates, _ := h.store.ListIncidentUpdates(ctx, id)
	enriched, err := incidentenrich.List(ctx, h.store, []model.Incident{*inc})
	if err != nil || len(enriched) == 0 {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load incident")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"incident": enriched[0],
		"updates":  updates,
	})
}
