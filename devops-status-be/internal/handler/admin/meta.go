package admin

import (
	"net/http"
	"strings"

	"github.com/devops-status/be/internal/handler"
)

// MetaHandler exposes read-only backend context for the admin UI (URLs differ in dev vs prod).
type MetaHandler struct {
	externalURL string
	isDev       bool
}

func NewMetaHandler(externalURL string, isDev bool) *MetaHandler {
	return &MetaHandler{externalURL: strings.TrimSpace(externalURL), isDev: isDev}
}

func (h *MetaHandler) Get(w http.ResponseWriter, r *http.Request) {
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"external_url": h.externalURL,
		"is_dev":       h.isDev,
	})
}
