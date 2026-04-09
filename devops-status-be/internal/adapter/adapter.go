package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type ProbeResult struct {
	Success   bool           `json:"success"`
	RawValue  float64        `json:"raw_value,omitempty"`
	LatencyMs int            `json:"latency_ms"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Trace     string         `json:"trace,omitempty"`
	Error     string         `json:"error,omitempty"`
	ProbedAt  time.Time      `json:"probed_at"`
}

type Adapter interface {
	Name() string
	Probe(ctx context.Context, config json.RawMessage) (*ProbeResult, error)
}

var registry = map[string]Adapter{}

func Register(a Adapter) {
	registry[a.Name()] = a
}

func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("adapter not found: %s", name)
	}
	return a, nil
}

func All() map[string]Adapter {
	return registry
}
