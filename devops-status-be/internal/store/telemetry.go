package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func telemetryDisplayNameArg(input CreateTelemetryInput) any {
	s := strings.TrimSpace(input.DisplayName)
	if s == "" {
		return nil
	}
	return s
}

func scanDisplayName(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func (s *Store) ListTelemetry(ctx context.Context) ([]model.Telemetry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, display_name, adapter, config_json, qos_thresholds, execution_target, secret_ref, timeout_ms, retries, created_at, updated_at
		 FROM telemetry ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list telemetry: %w", err)
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

// TelemetryNameExists reports whether another row already uses this name (excluding excludeID when set).
func (s *Store) TelemetryNameExists(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	var found uuid.UUID
	var err error
	if excludeID != nil {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM telemetry WHERE LOWER(name) = LOWER($1) AND id <> $2 LIMIT 1`,
			name, *excludeID).Scan(&found)
	} else {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM telemetry WHERE LOWER(name) = LOWER($1) LIMIT 1`,
			name).Scan(&found)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("telemetry name lookup: %w", err)
	}
	return true, nil
}

func (s *Store) GetTelemetryByID(ctx context.Context, id uuid.UUID) (*model.Telemetry, error) {
	var t model.Telemetry
	var cfgBytes, qosBytes []byte
	var dn sql.NullString
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, display_name, adapter, config_json, qos_thresholds, execution_target, secret_ref, timeout_ms, retries, created_at, updated_at
		 FROM telemetry WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &dn, &t.Adapter, &cfgBytes, &qosBytes, &t.ExecutionTarget, &t.SecretRef, &t.TimeoutMs, &t.Retries, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get telemetry: %w", err)
	}
	t.DisplayName = scanDisplayName(dn)
	_ = json.Unmarshal(cfgBytes, &t.ConfigJSON)
	_ = json.Unmarshal(qosBytes, &t.QosThresholds)
	return &t, nil
}

type CreateTelemetryInput struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Adapter         string `json:"adapter"`
	ConfigJSON      any    `json:"config_json"`
	QosThresholds   any    `json:"qos_thresholds"`
	ExecutionTarget string `json:"execution_target"`
	SecretRef       string `json:"secret_ref"`
	TimeoutMs       int    `json:"timeout_ms"`
	Retries         int    `json:"retries"`
}

func (s *Store) CreateTelemetry(ctx context.Context, input CreateTelemetryInput) (*model.Telemetry, error) {
	cfgBytes, _ := json.Marshal(input.ConfigJSON)
	qosBytes, _ := json.Marshal(input.QosThresholds)
	if len(qosBytes) == 0 || string(qosBytes) == "null" {
		qosBytes = []byte("{}")
	}
	execTarget := input.ExecutionTarget
	if execTarget == "" {
		execTarget = "backend"
	}
	var t model.Telemetry
	var cfgOut, qosOut []byte
	var dn sql.NullString
	dnArg := telemetryDisplayNameArg(input)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO telemetry (name, display_name, adapter, config_json, qos_thresholds, execution_target, secret_ref, timeout_ms, retries)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, name, display_name, adapter, config_json, qos_thresholds, execution_target, secret_ref, timeout_ms, retries, created_at, updated_at`,
		input.Name, dnArg, input.Adapter, cfgBytes, qosBytes, execTarget, input.SecretRef, input.TimeoutMs, input.Retries).
		Scan(&t.ID, &t.Name, &dn, &t.Adapter, &cfgOut, &qosOut, &t.ExecutionTarget, &t.SecretRef, &t.TimeoutMs, &t.Retries, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create telemetry: %w", err)
	}
	t.DisplayName = scanDisplayName(dn)
	_ = json.Unmarshal(cfgOut, &t.ConfigJSON)
	_ = json.Unmarshal(qosOut, &t.QosThresholds)
	return &t, nil
}

func (s *Store) UpdateTelemetry(ctx context.Context, id uuid.UUID, input CreateTelemetryInput) (*model.Telemetry, error) {
	cfgBytes, _ := json.Marshal(input.ConfigJSON)
	qosBytes, _ := json.Marshal(input.QosThresholds)
	if len(qosBytes) == 0 || string(qosBytes) == "null" {
		qosBytes = []byte("{}")
	}
	execTarget := input.ExecutionTarget
	if execTarget == "" {
		execTarget = "backend"
	}
	var t model.Telemetry
	var cfgOut, qosOut []byte
	var dn sql.NullString
	dnArg := telemetryDisplayNameArg(input)
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE telemetry SET name=$1, display_name=$2, adapter=$3, config_json=$4, qos_thresholds=$5, execution_target=$6, secret_ref=$7, timeout_ms=$8, retries=$9, updated_at=$10
		 WHERE id=$11
		 RETURNING id, name, display_name, adapter, config_json, qos_thresholds, execution_target, secret_ref, timeout_ms, retries, created_at, updated_at`,
		input.Name, dnArg, input.Adapter, cfgBytes, qosBytes, execTarget, input.SecretRef, input.TimeoutMs, input.Retries, now, id).
		Scan(&t.ID, &t.Name, &dn, &t.Adapter, &cfgOut, &qosOut, &t.ExecutionTarget, &t.SecretRef, &t.TimeoutMs, &t.Retries, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update telemetry: %w", err)
	}
	t.DisplayName = scanDisplayName(dn)
	_ = json.Unmarshal(cfgOut, &t.ConfigJSON)
	_ = json.Unmarshal(qosOut, &t.QosThresholds)
	return &t, nil
}

func (s *Store) DeleteTelemetry(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM telemetry WHERE id = $1`, id)
	return err
}
