package public

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type HistoryHandler struct {
	store *store.Store
}

func NewHistoryHandler(s *store.Store) *HistoryHandler {
	return &HistoryHandler{store: s}
}

func (h *HistoryHandler) ServiceHistory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.store.GetServiceBySlug(r.Context(), slug)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	rollups, err := h.store.GetDailyRollups(r.Context(), "service", svc.ID, 90)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load history")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"service": svc,
		"days":    rollups,
	})
}

func (h *HistoryHandler) EnvironmentHistory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env, err := h.store.GetEnvironmentBySlug(r.Context(), slug)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}

	rollups, err := h.store.GetDailyRollups(r.Context(), "environment", env.ID, 90)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load history")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"days":        rollups,
	})
}
