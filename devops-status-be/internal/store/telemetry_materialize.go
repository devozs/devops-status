package store

import (
	"context"
	"encoding/json"

	"github.com/devops-status/be/internal/secrets"
	"github.com/google/uuid"
)

// MaterializeTelemetryConfig is retained for API compatibility; Prometheus telemetry uses service providers only (see ResolveTelemetryProbeConfig).
func (s *Store) MaterializeTelemetryConfig(ctx context.Context, sec *secrets.Store, telemetryID uuid.UUID, adapter string, raw json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	_ = sec
	_ = telemetryID
	_ = adapter
	return raw, nil
}

// MaterializePrometheusConfigWithCredentials resolves credentials without persisting telemetry id (probe test draft).
func MaterializePrometheusConfigWithCredentials(raw json.RawMessage, creds secrets.PrometheusCredentials) (json.RawMessage, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return raw, err
	}
	if creds.BearerToken != "" {
		cfg["bearer_token"] = creds.BearerToken
	}
	if creds.Username != "" {
		cfg["username"] = creds.Username
	}
	if creds.Password != "" {
		cfg["password"] = creds.Password
	}
	return json.Marshal(cfg)
}
