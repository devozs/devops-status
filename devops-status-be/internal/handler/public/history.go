package public

import (
	"net/http"
	"time"

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

	endPtr, err := historyEndExclusiveParam(r)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid end")
		return
	}
	rollups, err := h.store.GetDailyRollups(r.Context(), "service", svc.ID, 90, endPtr)
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

	endPtr, err := historyEndExclusiveParam(r)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid end")
		return
	}
	rollups, err := h.store.GetDailyRollups(r.Context(), "environment", env.ID, 90, endPtr)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load history")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"days":        rollups,
	})
}

// historyEndExclusiveParam returns exclusive UTC end instant for rollup window (samples with day <= end date).
// Query end=YYYY-MM-DD → exclusive midnight at start of the next calendar day in UTC.
func historyEndExclusiveParam(r *http.Request) (*time.Time, error) {
	v := r.URL.Query().Get("end")
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, time.UTC)
	if err != nil {
		return nil, err
	}
	excl := t.AddDate(0, 0, 1)
	return &excl, nil
}
