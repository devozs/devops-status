package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

// ListTelemetryReferencingShellHint returns telemetry rows whose config_json references hintID in any shell-hint slot
// (top-level for kubernetes/cli, or under kubernetes for liveness).
func (s *Store) ListTelemetryReferencingShellHint(ctx context.Context, hintID uuid.UUID) ([]model.Telemetry, error) {
	idStr := hintID.String()
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, display_name, adapter, config_json, qos_thresholds, execution_target, secret_ref, timeout_ms, retries, created_at, updated_at
		FROM telemetry
		WHERE (config_json::jsonb ->> 'job_env_hint_id') = $1
		   OR (config_json::jsonb ->> 'job_node_selector_hint_id') = $1
		   OR (config_json::jsonb ->> 'container_prep_hint_id') = $1
		   OR (config_json::jsonb ->> 'probe_shell_hint_id') = $1
		   OR (config_json::jsonb #>> '{kubernetes,job_env_hint_id}') = $1
		   OR (config_json::jsonb #>> '{kubernetes,job_node_selector_hint_id}') = $1
		   OR (config_json::jsonb #>> '{kubernetes,container_prep_hint_id}') = $1
		   OR (config_json::jsonb #>> '{kubernetes,probe_shell_hint_id}') = $1
		ORDER BY name ASC`, idStr)
	if err != nil {
		return nil, fmt.Errorf("list telemetry by shell hint ref: %w", err)
	}
	defer rows.Close()

	out := make([]model.Telemetry, 0)
	for rows.Next() {
		var t model.Telemetry
		var cfgBytes, qosBytes []byte
		var dn sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &dn, &t.Adapter, &cfgBytes, &qosBytes, &t.ExecutionTarget, &t.SecretRef, &t.TimeoutMs, &t.Retries, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan telemetry: %w", err)
		}
		t.DisplayName = scanDisplayName(dn)
		_ = json.Unmarshal(cfgBytes, &t.ConfigJSON)
		_ = json.Unmarshal(qosBytes, &t.QosThresholds)
		out = append(out, t)
	}
	return out, nil
}
