package liveness

import (
	"context"
	"encoding/json"

	"github.com/devops-status/be/internal/adapter"
)

func probePrometheus(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	return adapter.NewPrometheusAdapter().Probe(ctx, cfg)
}
