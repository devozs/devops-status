package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/handler"
)

// ensure context is used
var _ context.Context

type ProbeTestHandler struct{}

func NewProbeTestHandler() *ProbeTestHandler {
	return &ProbeTestHandler{}
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

	cfgBytes, _ := json.Marshal(req.ConfigJSON)

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
