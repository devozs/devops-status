package adapter

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func imgs() CLIRunnerImages {
	return CLIRunnerImages{Alpine: "a:1", Ubuntu24: "u:1", Legacy: "l:1"}
}

func TestKubernetesAdapter_legacyMissingEndpoint(t *testing.T) {
	a := NewKubernetesAdapter(imgs(), nil)
	raw, _ := json.Marshal(KubernetesConfig{
		CheckType: "api_health",
		Endpoint:  "",
	})
	r, err := a.Probe(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.Success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(r.Error, "endpoint") {
		t.Fatalf("error: %q", r.Error)
	}
}

func TestKubernetesAdapter_shellMissingK8sRun(t *testing.T) {
	a := NewKubernetesAdapter(imgs(), nil)
	raw, _ := json.Marshal(KubernetesConfig{
		CLIShell: "echo 1",
		Endpoint: "https://example",
		Token:    "t",
	})
	r, err := a.Probe(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.Success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(r.Error, "shell runner not configured") {
		t.Fatalf("error: %q", r.Error)
	}
}

func TestKubernetesAdapter_shellMissingToken(t *testing.T) {
	run := func(context.Context, string, string, bool, string, CLIConfig) (*ProbeResult, error) {
		return nil, nil
	}
	a := NewKubernetesAdapter(imgs(), run)
	raw, _ := json.Marshal(KubernetesConfig{
		CLIShell: "echo 1",
		Endpoint: "https://example",
		Token:    "",
	})
	r, err := a.Probe(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.Success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(r.Error, "token required") {
		t.Fatalf("error: %q", r.Error)
	}
}
