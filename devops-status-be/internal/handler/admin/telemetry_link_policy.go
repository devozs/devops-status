package admin

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
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

// TelemetryClusterID returns the Kubernetes cluster id from telemetry config when the probe is cluster-scoped.
func TelemetryClusterID(t *model.Telemetry) (uuid.UUID, bool) {
	if t == nil {
		return uuid.Nil, false
	}
	raw, err := json.Marshal(t.ConfigJSON)
	if err != nil {
		return uuid.Nil, false
	}
	switch strings.ToLower(strings.TrimSpace(t.Adapter)) {
	case "kubernetes":
		var c struct {
			ClusterID string `json:"cluster_id"`
		}
		if json.Unmarshal(raw, &c) != nil {
			return uuid.Nil, false
		}
		id, err := uuid.Parse(strings.TrimSpace(c.ClusterID))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, false
		}
		return id, true
	case "liveness":
		var c struct {
			Source     string          `json:"source"`
			Kubernetes json.RawMessage `json:"kubernetes"`
		}
		if json.Unmarshal(raw, &c) != nil {
			return uuid.Nil, false
		}
		if strings.ToLower(strings.TrimSpace(c.Source)) != "kubernetes" || len(c.Kubernetes) == 0 {
			return uuid.Nil, false
		}
		var inner struct {
			ClusterID string `json:"cluster_id"`
		}
		if json.Unmarshal(c.Kubernetes, &inner) != nil {
			return uuid.Nil, false
		}
		id, err := uuid.Parse(strings.TrimSpace(inner.ClusterID))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, false
		}
		return id, true
	default:
		return uuid.Nil, false
	}
}

// TelemetryLinkedServiceProviderID returns the service_provider_id from telemetry config for SP-backed probes.
func TelemetryLinkedServiceProviderID(t *model.Telemetry) (uuid.UUID, bool) {
	if t == nil {
		return uuid.Nil, false
	}
	raw, err := json.Marshal(t.ConfigJSON)
	if err != nil {
		return uuid.Nil, false
	}
	adapter := strings.ToLower(strings.TrimSpace(t.Adapter))
	switch adapter {
	case "http", "prometheus", "hlctl":
		var c struct {
			ServiceProviderID string `json:"service_provider_id"`
		}
		if json.Unmarshal(raw, &c) != nil {
			return uuid.Nil, false
		}
		id, err := uuid.Parse(strings.TrimSpace(c.ServiceProviderID))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, false
		}
		return id, true
	case "liveness":
		if LivenessTelemetrySource(t.ConfigJSON) != "service_provider" {
			return uuid.Nil, false
		}
		var c struct {
			ServiceProviderID string `json:"service_provider_id"`
		}
		if json.Unmarshal(raw, &c) != nil {
			return uuid.Nil, false
		}
		id, err := uuid.Parse(strings.TrimSpace(c.ServiceProviderID))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, false
		}
		return id, true
	default:
		return uuid.Nil, false
	}
}

// ValidateEnvironmentTelemetryInfra ensures the environment has a primary cluster and telemetry targets the same cluster.
func ValidateEnvironmentTelemetryInfra(env *model.Environment, tel *model.Telemetry) error {
	if env == nil || tel == nil {
		return errors.New("invalid environment or telemetry")
	}
	if env.K8sClusterID == nil || *env.K8sClusterID == uuid.Nil {
		return errors.New("environment must have a primary kubernetes cluster before adding telemetry links")
	}
	tcid, ok := TelemetryClusterID(tel)
	if !ok {
		return errors.New("telemetry must specify a kubernetes cluster that matches this environment")
	}
	if tcid != *env.K8sClusterID {
		return errors.New("telemetry kubernetes cluster must match the environment primary cluster")
	}
	return nil
}

// ValidateServiceTelemetryInfra ensures the service has a provider and telemetry references the same provider (CLI excepted).
func ValidateServiceTelemetryInfra(svc *model.Service, tel *model.Telemetry) error {
	if svc == nil || tel == nil {
		return errors.New("invalid service or telemetry")
	}
	if svc.ServiceProviderID == nil || *svc.ServiceProviderID == uuid.Nil {
		return errors.New("service must have a service provider before adding telemetry links")
	}
	if strings.EqualFold(strings.TrimSpace(tel.Adapter), "cli") {
		return nil
	}
	spid, ok := TelemetryLinkedServiceProviderID(tel)
	if !ok {
		return errors.New("telemetry must reference a service provider that matches this service")
	}
	if spid != *svc.ServiceProviderID {
		return errors.New("telemetry service provider must match the service's service provider")
	}
	return nil
}
