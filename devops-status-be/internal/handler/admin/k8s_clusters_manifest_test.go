package admin

import (
	"strings"
	"testing"
)

func TestGenerateShellProbeRBACManifest(t *testing.T) {
	y := generateShellProbeRBACManifest()
	if !strings.Contains(y, "name: devops-status-system") || !strings.Contains(y, "name: devops-status-agent") {
		t.Fatalf("expected namespace and service account: %q", y)
	}
	if strings.Contains(y, "kind: Job") {
		t.Fatal("shell RBAC manifest must not include Job")
	}
}

func TestGenerateOnboardingManifestIncludesShellRBAC(t *testing.T) {
	full := generateOnboardingManifest("550e8400-e29b-41d4-a716-446655440000", "handshake-token-value", "https://status.example.com")
	base := generateShellProbeRBACManifest()
	if !strings.HasPrefix(full, base) {
		t.Fatal("onboarding manifest should start with shell RBAC documents")
	}
	if !strings.Contains(full, "kind: Job") {
		t.Fatal("onboarding manifest should include Job")
	}
}
