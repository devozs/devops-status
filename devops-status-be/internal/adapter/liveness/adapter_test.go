package liveness

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAdapterProbe_invalidJSON(t *testing.T) {
	a := NewAdapter()
	_, err := a.Probe(context.Background(), json.RawMessage(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAdapterProbe_missingLivenessCheck(t *testing.T) {
	a := NewAdapter()
	_, err := a.Probe(context.Background(), json.RawMessage(`{"config":{}}`))
	if err == nil || !strings.Contains(err.Error(), "liveness_check") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAdapterProbe_unsupportedCheck(t *testing.T) {
	a := NewAdapter()
	raw, _ := json.Marshal(map[string]any{
		"liveness_check": "unknown_kind",
		"config":         map[string]any{},
	})
	_, err := a.Probe(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "unsupported liveness_check") {
		t.Fatalf("unexpected err: %v", err)
	}
}
