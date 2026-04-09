package store

import (
	"context"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) ListIncidents(ctx context.Context, limit int, targetType string) ([]model.Incident, error) {
	query := `SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at, created_at, updated_at
	          FROM incidents`
	args := []any{}
	argIdx := 1

	if targetType != "" {
		query += fmt.Sprintf(` WHERE target_type = $%d`, argIdx)
		args = append(args, targetType)
		argIdx++
	}

	query += ` ORDER BY started_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []model.Incident
	for rows.Next() {
		var inc model.Incident
		if err := rows.Scan(&inc.ID, &inc.TargetType, &inc.TargetID, &inc.Title, &inc.Status, &inc.Severity,
			&inc.StartedAt, &inc.ResolvedAt, &inc.CreatedAt, &inc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

func (s *Store) GetIncidentByID(ctx context.Context, id uuid.UUID) (*model.Incident, error) {
	var inc model.Incident
	err := s.pool.QueryRow(ctx,
		`SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at, created_at, updated_at
		 FROM incidents WHERE id=$1`, id).
		Scan(&inc.ID, &inc.TargetType, &inc.TargetID, &inc.Title, &inc.Status, &inc.Severity,
			&inc.StartedAt, &inc.ResolvedAt, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}
	return &inc, nil
}

func (s *Store) GetOpenIncident(ctx context.Context, targetType string, targetID uuid.UUID) (*model.Incident, error) {
	var inc model.Incident
	err := s.pool.QueryRow(ctx,
		`SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at, created_at, updated_at
		 FROM incidents WHERE target_type=$1 AND target_id=$2 AND resolved_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`, targetType, targetID).
		Scan(&inc.ID, &inc.TargetType, &inc.TargetID, &inc.Title, &inc.Status, &inc.Severity,
			&inc.StartedAt, &inc.ResolvedAt, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

func (s *Store) CreateIncident(ctx context.Context, targetType string, targetID uuid.UUID, title, severity string) (*model.Incident, error) {
	var inc model.Incident
	err := s.pool.QueryRow(ctx,
		`INSERT INTO incidents (target_type, target_id, title, status, severity)
		 VALUES ($1, $2, $3, 'investigating', $4)
		 RETURNING id, target_type, target_id, title, status, severity, started_at, resolved_at, created_at, updated_at`,
		targetType, targetID, title, severity).
		Scan(&inc.ID, &inc.TargetType, &inc.TargetID, &inc.Title, &inc.Status, &inc.Severity,
			&inc.StartedAt, &inc.ResolvedAt, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create incident: %w", err)
	}
	return &inc, nil
}

func (s *Store) UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string) error {
	now := time.Now()
	var resolvedAt *time.Time
	if status == "resolved" {
		resolvedAt = &now
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE incidents SET status=$1, resolved_at=$2, updated_at=$3 WHERE id=$4`,
		status, resolvedAt, now, id)
	return err
}

func (s *Store) CreateIncidentUpdate(ctx context.Context, incidentID uuid.UUID, status, message, author string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO incident_updates (incident_id, status, message, author) VALUES ($1, $2, $3, $4)`,
		incidentID, status, message, author)
	return err
}

func (s *Store) ListIncidentUpdates(ctx context.Context, incidentID uuid.UUID) ([]model.IncidentUpdate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, incident_id, status, message, author, created_at
		 FROM incident_updates WHERE incident_id=$1 ORDER BY created_at DESC`, incidentID)
	if err != nil {
		return nil, fmt.Errorf("list incident updates: %w", err)
	}
	defer rows.Close()

	var updates []model.IncidentUpdate
	for rows.Next() {
		var u model.IncidentUpdate
		if err := rows.Scan(&u.ID, &u.IncidentID, &u.Status, &u.Message, &u.Author, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan incident update: %w", err)
		}
		updates = append(updates, u)
	}
	return updates, nil
}
