package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type ProbeTestHandler struct {
	store          *store.Store
	k8sInsecureTLS bool
}

func NewProbeTestHandler(s *store.Store, k8sInsecureTLS bool) *ProbeTestHandler {
	return &ProbeTestHandler{store: s, k8sInsecureTLS: k8sInsecureTLS}
}

type probeTestRequest struct {
	Adapter    string `json:"adapter"`
	ConfigJSON any    `json:"config_json"`
}

func (h *ProbeTestHandler) Test(w http.ResponseWriter, r *http.Request) {
	var req probeTestRequest
	if err := handler.DecodeJSON(r, &req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	a, err := adapter.Get(req.Adapter)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "unsupported adapter: "+req.Adapter)
		return
	}

	cfgBytes, err := json.Marshal(req.ConfigJSON)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid config_json")
		return
	}
	cfgBytes, err = h.store.MergeKubernetesProbeConfig(r.Context(), req.Adapter, cfgBytes, h.k8sInsecureTLS)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to merge k8s config")
		return
	}

	ctx := r.Context()
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := a.Probe(testCtx, cfgBytes)
	if err != nil {
		handler.WriteJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	handler.WriteJSON(w, http.StatusOK, result)
}
