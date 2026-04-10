package store

import (
	"context"
	"fmt"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) ListServiceProbeBindings(ctx context.Context, serviceID *uuid.UUID) ([]model.ServiceProbeBinding, error) {
	query := `SELECT id, service_id, probe_kind, data_source_id, sample_interval_sec, window_size,
	          consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at
	          FROM service_probe_bindings`
	args := []any{}
	if serviceID != nil {
		query += ` WHERE service_id = $1`
		args = append(args, *serviceID)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list service bindings: %w", err)
	}
	defer rows.Close()

	bindings := make([]model.ServiceProbeBinding, 0)
	for rows.Next() {
		var b model.ServiceProbeBinding
		if err := rows.Scan(&b.ID, &b.ServiceID, &b.ProbeKind, &b.DataSourceID, &b.SampleIntervalSec,
			&b.WindowSize, &b.ConsecutiveFailuresToDown, &b.ConsecutiveSuccessToRecover, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan service binding: %w", err)
		}
		bindings = append(bindings, b)
	}
	return bindings, nil
}

func (s *Store) ListEnvironmentProbeBindings(ctx context.Context, envID *uuid.UUID) ([]model.EnvironmentProbeBinding, error) {
	query := `SELECT id, environment_id, probe_kind, data_source_id, sample_interval_sec, window_size,
	          consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at
	          FROM environment_probe_bindings`
	args := []any{}
	if envID != nil {
		query += ` WHERE environment_id = $1`
		args = append(args, *envID)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list environment bindings: %w", err)
	}
	defer rows.Close()

	bindings := make([]model.EnvironmentProbeBinding, 0)
	for rows.Next() {
		var b model.EnvironmentProbeBinding
		if err := rows.Scan(&b.ID, &b.EnvironmentID, &b.ProbeKind, &b.DataSourceID, &b.SampleIntervalSec,
			&b.WindowSize, &b.ConsecutiveFailuresToDown, &b.ConsecutiveSuccessToRecover, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan env binding: %w", err)
		}
		bindings = append(bindings, b)
	}
	return bindings, nil
}

type CreateBindingInput struct {
	ProbeKind                    string `json:"probe_kind"`
	DataSourceID                 string `json:"data_source_id"`
	SampleIntervalSec            int    `json:"sample_interval_sec"`
	WindowSize                   int    `json:"window_size"`
	ConsecutiveFailuresToDown    int    `json:"consecutive_failures_to_down"`
	ConsecutiveSuccessToRecover  int    `json:"consecutive_success_to_recover"`
}

func (s *Store) CreateServiceProbeBinding(ctx context.Context, serviceID uuid.UUID, input CreateBindingInput) (*model.ServiceProbeBinding, error) {
	dsID, _ := uuid.Parse(input.DataSourceID)
	var b model.ServiceProbeBinding
	err := s.pool.QueryRow(ctx,
		`INSERT INTO service_probe_bindings (service_id, probe_kind, data_source_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, service_id, probe_kind, data_source_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at`,
		serviceID, input.ProbeKind, dsID, input.SampleIntervalSec, input.WindowSize, input.ConsecutiveFailuresToDown, input.ConsecutiveSuccessToRecover).
		Scan(&b.ID, &b.ServiceID, &b.ProbeKind, &b.DataSourceID, &b.SampleIntervalSec, &b.WindowSize, &b.ConsecutiveFailuresToDown, &b.ConsecutiveSuccessToRecover, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create service binding: %w", err)
	}
	return &b, nil
}

func (s *Store) CreateEnvironmentProbeBinding(ctx context.Context, envID uuid.UUID, input CreateBindingInput) (*model.EnvironmentProbeBinding, error) {
	dsID, _ := uuid.Parse(input.DataSourceID)
	var b model.EnvironmentProbeBinding
	err := s.pool.QueryRow(ctx,
		`INSERT INTO environment_probe_bindings (environment_id, probe_kind, data_source_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, environment_id, probe_kind, data_source_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at`,
		envID, input.ProbeKind, dsID, input.SampleIntervalSec, input.WindowSize, input.ConsecutiveFailuresToDown, input.ConsecutiveSuccessToRecover).
		Scan(&b.ID, &b.EnvironmentID, &b.ProbeKind, &b.DataSourceID, &b.SampleIntervalSec, &b.WindowSize, &b.ConsecutiveFailuresToDown, &b.ConsecutiveSuccessToRecover, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create env binding: %w", err)
	}
	return &b, nil
}

func (s *Store) DeleteServiceProbeBinding(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM service_probe_bindings WHERE id = $1`, id)
	return err
}

func (s *Store) DeleteEnvironmentProbeBinding(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM environment_probe_bindings WHERE id = $1`, id)
	return err
}
