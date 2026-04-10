package store

import (
	"context"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) ListServiceTelemetryLinks(ctx context.Context, serviceID *uuid.UUID) ([]model.ServiceTelemetryLink, error) {
	query := `SELECT id, service_id, telemetry_id, sample_interval_sec, window_size,
		consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at
	          FROM service_telemetry_links`
	args := []any{}
	if serviceID != nil {
		query += ` WHERE service_id = $1`
		args = append(args, *serviceID)
	}
	query += ` ORDER BY service_id, id`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list service telemetry links: %w", err)
	}
	defer rows.Close()

	out := make([]model.ServiceTelemetryLink, 0)
	for rows.Next() {
		var l model.ServiceTelemetryLink
		if err := rows.Scan(&l.ID, &l.ServiceID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan service telemetry link: %w", err)
		}
		out = append(out, l)
	}
	return out, nil
}

func (s *Store) ListEnvironmentTelemetryLinks(ctx context.Context, envID *uuid.UUID) ([]model.EnvironmentTelemetryLink, error) {
	query := `SELECT id, environment_id, telemetry_id, sample_interval_sec, window_size,
		consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at
	          FROM environment_telemetry_links`
	args := []any{}
	if envID != nil {
		query += ` WHERE environment_id = $1`
		args = append(args, *envID)
	}
	query += ` ORDER BY environment_id, id`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list environment telemetry links: %w", err)
	}
	defer rows.Close()

	out := make([]model.EnvironmentTelemetryLink, 0)
	for rows.Next() {
		var l model.EnvironmentTelemetryLink
		if err := rows.Scan(&l.ID, &l.EnvironmentID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan environment telemetry link: %w", err)
		}
		out = append(out, l)
	}
	return out, nil
}

type CreateTelemetryLinkInput struct {
	TelemetryID                 uuid.UUID `json:"telemetry_id"`
	SampleIntervalSec           int       `json:"sample_interval_sec"`
	WindowSize                  int       `json:"window_size"`
	ConsecutiveFailuresToDown   int       `json:"consecutive_failures_to_down"`
	ConsecutiveSuccessToRecover int       `json:"consecutive_success_to_recover"`
}

func (s *Store) CreateServiceTelemetryLink(ctx context.Context, serviceID uuid.UUID, input CreateTelemetryLinkInput) (*model.ServiceTelemetryLink, error) {
	interval, window, failDown, succUp := telemetryLinkDefaults(input)
	var l model.ServiceTelemetryLink
	err := s.pool.QueryRow(ctx,
		`INSERT INTO service_telemetry_links (service_id, telemetry_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, service_id, telemetry_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at`,
		serviceID, input.TelemetryID, interval, window, failDown, succUp).
		Scan(&l.ID, &l.ServiceID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create service telemetry link: %w", err)
	}
	return &l, nil
}

func (s *Store) CreateEnvironmentTelemetryLink(ctx context.Context, envID uuid.UUID, input CreateTelemetryLinkInput) (*model.EnvironmentTelemetryLink, error) {
	interval, window, failDown, succUp := telemetryLinkDefaults(input)
	var l model.EnvironmentTelemetryLink
	err := s.pool.QueryRow(ctx,
		`INSERT INTO environment_telemetry_links (environment_id, telemetry_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, environment_id, telemetry_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at`,
		envID, input.TelemetryID, interval, window, failDown, succUp).
		Scan(&l.ID, &l.EnvironmentID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create environment telemetry link: %w", err)
	}
	return &l, nil
}

func telemetryLinkDefaults(input CreateTelemetryLinkInput) (interval, window, failDown, succUp int) {
	interval = input.SampleIntervalSec
	if interval <= 0 {
		interval = 60
	}
	window = input.WindowSize
	if window <= 0 {
		window = 5
	}
	failDown = input.ConsecutiveFailuresToDown
	if failDown <= 0 {
		failDown = 3
	}
	succUp = input.ConsecutiveSuccessToRecover
	if succUp <= 0 {
		succUp = 2
	}
	return interval, window, failDown, succUp
}

func (s *Store) DeleteServiceTelemetryLink(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM service_telemetry_links WHERE id = $1`, id)
	return err
}

func (s *Store) DeleteEnvironmentTelemetryLink(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM environment_telemetry_links WHERE id = $1`, id)
	return err
}

func (s *Store) GetServiceTelemetryLinkByID(ctx context.Context, id uuid.UUID) (*model.ServiceTelemetryLink, error) {
	var l model.ServiceTelemetryLink
	err := s.pool.QueryRow(ctx,
		`SELECT id, service_id, telemetry_id, sample_interval_sec, window_size,
			consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at
		 FROM service_telemetry_links WHERE id = $1`, id).
		Scan(&l.ID, &l.ServiceID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get service telemetry link: %w", err)
	}
	return &l, nil
}

func (s *Store) GetEnvironmentTelemetryLinkByID(ctx context.Context, id uuid.UUID) (*model.EnvironmentTelemetryLink, error) {
	var l model.EnvironmentTelemetryLink
	err := s.pool.QueryRow(ctx,
		`SELECT id, environment_id, telemetry_id, sample_interval_sec, window_size,
			consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at
		 FROM environment_telemetry_links WHERE id = $1`, id).
		Scan(&l.ID, &l.EnvironmentID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get environment telemetry link: %w", err)
	}
	return &l, nil
}

func (s *Store) UpdateServiceTelemetryLink(ctx context.Context, id uuid.UUID, input CreateTelemetryLinkInput) (*model.ServiceTelemetryLink, error) {
	interval, window, failDown, succUp := telemetryLinkDefaults(input)
	var l model.ServiceTelemetryLink
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE service_telemetry_links SET telemetry_id=$1, sample_interval_sec=$2, window_size=$3,
			consecutive_failures_to_down=$4, consecutive_success_to_recover=$5, updated_at=$6
		 WHERE id=$7
		 RETURNING id, service_id, telemetry_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at`,
		input.TelemetryID, interval, window, failDown, succUp, now, id).
		Scan(&l.ID, &l.ServiceID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update service telemetry link: %w", err)
	}
	return &l, nil
}

func (s *Store) UpdateEnvironmentTelemetryLink(ctx context.Context, id uuid.UUID, input CreateTelemetryLinkInput) (*model.EnvironmentTelemetryLink, error) {
	interval, window, failDown, succUp := telemetryLinkDefaults(input)
	var l model.EnvironmentTelemetryLink
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE environment_telemetry_links SET telemetry_id=$1, sample_interval_sec=$2, window_size=$3,
			consecutive_failures_to_down=$4, consecutive_success_to_recover=$5, updated_at=$6
		 WHERE id=$7
		 RETURNING id, environment_id, telemetry_id, sample_interval_sec, window_size, consecutive_failures_to_down, consecutive_success_to_recover, created_at, updated_at`,
		input.TelemetryID, interval, window, failDown, succUp, now, id).
		Scan(&l.ID, &l.EnvironmentID, &l.TelemetryID, &l.SampleIntervalSec, &l.WindowSize,
			&l.ConsecutiveFailuresToDown, &l.ConsecutiveSuccessToRecover, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update environment telemetry link: %w", err)
	}
	return &l, nil
}
