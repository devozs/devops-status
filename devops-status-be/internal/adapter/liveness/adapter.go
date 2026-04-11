package liveness

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devops-status/be/internal/adapter"
)

// Adapter implements adapter.Adapter for generic liveness probes.
type Adapter struct{}

func NewAdapter() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "liveness" }

func (a *Adapter) Probe(ctx context.Context, configRaw json.RawMessage) (*adapter.ProbeResult, error) {
	var w ResolvedWire
	if err := json.Unmarshal(configRaw, &w); err != nil {
		return nil, fmt.Errorf("parse liveness resolved config: %w", err)
	}
	k := strings.ToLower(strings.TrimSpace(w.LivenessCheck))
	if k == "" {
		return nil, fmt.Errorf("liveness_check is required")
	}

	switch k {
	case CheckKubernetes:
		return probeKubernetes(ctx, w.Config)
	case CheckPrometheus:
		return probePrometheus(ctx, w.Config)
	case CheckGrafana:
		return probeGrafana(ctx, w.Config)
	case CheckElasticsearch:
		return probeElasticsearch(ctx, w.Config)
	case CheckJenkins:
		return probeJenkins(ctx, w.Config)
	case CheckArtifactory:
		return probeArtifactory(ctx, w.Config)
	default:
		return nil, fmt.Errorf("unsupported liveness_check: %s", w.LivenessCheck)
	}
}
