package admin

import (
	"net/http"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type TelemetryLinksSummaryHandler struct {
	store *store.Store
}

func NewTelemetryLinksSummaryHandler(s *store.Store) *TelemetryLinksSummaryHandler {
	return &TelemetryLinksSummaryHandler{store: s}
}

// ListAll returns all service and environment telemetry links (for admin dashboard).
func (h *TelemetryLinksSummaryHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	svc, err := h.store.ListServiceTelemetryLinks(r.Context(), nil)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list service links")
		return
	}
	env, err := h.store.ListEnvironmentTelemetryLinks(r.Context(), nil)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list environment links")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"service_links":      svc,
		"environment_links": env,
	})
}
