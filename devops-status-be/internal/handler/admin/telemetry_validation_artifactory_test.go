package admin

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateTelemetryConfig_LivenessArtifactoryRequiresValueQoS(t *testing.T) {
	cfg := map[string]any{
		"source":              "service_provider",
		"service_provider_id": "550e8400-e29b-41d4-a716-446655440000",
		"artifactory": map[string]any{
			"download_repository_path": "libs-local/foo/bar.jar",
		},
	}
	cfgBytes, _ := json.Marshal(cfg)
	qos := map[string]any{
		"latency": map[string]any{"green_max_ms": 100, "yellow_max_ms": 200, "red_max_ms": 300},
	}
	qosBytes, _ := json.Marshal(qos)
	errs := ValidateTelemetryConfig("liveness", cfgBytes, qosBytes, "backend")
	if len(errs) == 0 {
		t.Fatal("expected errors")
	}
	ok := false
	for _, e := range errs {
		if strings.Contains(e, "artifactory download bandwidth requires qos_thresholds.value") {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("unexpected errs: %v", errs)
	}
}

func TestValidateTelemetryConfig_LivenessArtifactoryOnKubernetesSource(t *testing.T) {
	cfg := map[string]any{
		"source": "kubernetes",
		"kubernetes": map[string]any{
			"cluster_id":    "550e8400-e29b-41d4-a716-446655440001",
			"k8s_version":   "1.30",
			"check_type":    "api_health",
			"namespace":     "default",
			"resource_name": "",
		},
		"artifactory": map[string]any{
			"download_repository_path": "x/y",
		},
	}
	cfgBytes, _ := json.Marshal(cfg)
	qos := map[string]any{
		"latency": map[string]any{"green_max_ms": 100, "yellow_max_ms": 200, "red_max_ms": 300},
	}
	qosBytes, _ := json.Marshal(qos)
	errs := ValidateTelemetryConfig("liveness", cfgBytes, qosBytes, "k8s_cluster")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "artifactory settings are only valid when source is service_provider") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected kubernetes+artifactory error, got %v", errs)
	}
}

func TestValidateTelemetryConfig_LivenessArtifactoryOK(t *testing.T) {
	cfg := map[string]any{
		"source":              "service_provider",
		"service_provider_id": "550e8400-e29b-41d4-a716-446655440000",
		"artifactory": map[string]any{
			"download_repository_path": "libs-local/foo/bar.jar",
			"download_max_bytes":       1024,
		},
	}
	cfgBytes, _ := json.Marshal(cfg)
	qos := map[string]any{
		"latency": map[string]any{"green_max_ms": 100, "yellow_max_ms": 200, "red_max_ms": 300},
		"value": map[string]any{
			"green_operator": "gte", "green_threshold": 1e6,
			"yellow_operator": "gte", "yellow_threshold": 1e5,
		},
	}
	qosBytes, _ := json.Marshal(qos)
	errs := ValidateTelemetryConfig("liveness", cfgBytes, qosBytes, "backend")
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
}
