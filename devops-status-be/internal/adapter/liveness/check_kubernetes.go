package liveness

import (
	"context"
	"encoding/json"

	"github.com/devops-status/be/internal/adapter"
)

func probeKubernetes(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	a, err := adapter.Get("kubernetes")
	if err != nil {
		return nil, err
	}
	return a.Probe(ctx, cfg)
}
