package admin

import (
	"testing"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func TestTelemetryClusterID(t *testing.T) {
	cid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tel := &model.Telemetry{
		Adapter:    "kubernetes",
		ConfigJSON: map[string]any{"cluster_id": cid.String()},
	}
	got, ok := TelemetryClusterID(tel)
	if !ok || got != cid {
		t.Fatalf("kubernetes: got %v ok=%v want %v", got, ok, cid)
	}

	tel2 := &model.Telemetry{
		Adapter: "liveness",
		ConfigJSON: map[string]any{
			"source":     "kubernetes",
			"kubernetes": map[string]any{"cluster_id": cid.String()},
		},
	}
	got2, ok2 := TelemetryClusterID(tel2)
	if !ok2 || got2 != cid {
		t.Fatalf("liveness k8s: got %v ok=%v want %v", got2, ok2, cid)
	}
}

func TestValidateEnvironmentTelemetryInfra(t *testing.T) {
	cid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	env := &model.Environment{K8sClusterID: &cid}
	tel := &model.Telemetry{
		Adapter:    "kubernetes",
		ConfigJSON: map[string]any{"cluster_id": cid.String()},
	}
	if err := ValidateEnvironmentTelemetryInfra(env, tel); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEnvironmentTelemetryInfra(&model.Environment{}, tel); err == nil {
		t.Fatal("expected error when env has no cluster")
	}
	other := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	if err := ValidateEnvironmentTelemetryInfra(&model.Environment{K8sClusterID: &other}, tel); err == nil {
		t.Fatal("expected error on cluster mismatch")
	}
}

func TestValidateServiceTelemetryInfra(t *testing.T) {
	sp := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	svc := &model.Service{ServiceProviderID: &sp}
	httpTel := &model.Telemetry{
		Adapter:    "http",
		ConfigJSON: map[string]any{"service_provider_id": sp.String(), "path": "/", "method": "GET"},
	}
	if err := ValidateServiceTelemetryInfra(svc, httpTel); err != nil {
		t.Fatal(err)
	}
	cliTel := &model.Telemetry{Adapter: "cli", ConfigJSON: map[string]any{"parse_json": false, "execution_target": "backend"}}
	if err := ValidateServiceTelemetryInfra(svc, cliTel); err != nil {
		t.Fatal(err)
	}
	wrong := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	if err := ValidateServiceTelemetryInfra(svc, &model.Telemetry{
		Adapter:    "http",
		ConfigJSON: map[string]any{"service_provider_id": wrong.String(), "path": "/", "method": "GET"},
	}); err == nil {
		t.Fatal("expected SP mismatch error")
	}
	if err := ValidateServiceTelemetryInfra(&model.Service{}, httpTel); err == nil {
		t.Fatal("expected error when service has no provider")
	}
}
