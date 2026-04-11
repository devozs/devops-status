package admin

import (
	"encoding/json"
	"strings"

	"github.com/devops-status/be/internal/model"
)

// LivenessTelemetrySource returns the lowercase source field from liveness config_json (kubernetes | service_provider).
func LivenessTelemetrySource(cfg any) string {
	b, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	var c struct {
		Source string `json:"source"`
	}
	if json.Unmarshal(b, &c) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(c.Source))
}

// EnvironmentTelemetryAdapterAllowed reports whether telemetry can be linked to an environment.
func EnvironmentTelemetryAdapterAllowed(t *model.Telemetry) bool {
	if t == nil {
		return false
	}
	if t.Adapter == "kubernetes" {
		return true
	}
	if t.Adapter == "liveness" && LivenessTelemetrySource(t.ConfigJSON) == "kubernetes" {
		return true
	}
	return false
}
