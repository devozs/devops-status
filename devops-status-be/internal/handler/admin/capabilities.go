package admin

import (
	"net/http"

	"github.com/devops-status/be/internal/handler"
)

// CapabilitiesHandler exposes feature flags for the admin UI (e.g. HLCTL when kubectl+hlctl exist in the management image).
type CapabilitiesHandler struct {
	hlctlEnabled bool
}

func NewCapabilitiesHandler(hlctlEnabled bool) *CapabilitiesHandler {
	return &CapabilitiesHandler{hlctlEnabled: hlctlEnabled}
}

func (h *CapabilitiesHandler) Get(w http.ResponseWriter, r *http.Request) {
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"hlctl_enabled": h.hlctlEnabled,
	})
}
