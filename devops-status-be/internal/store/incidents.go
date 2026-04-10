package store

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

var qosPassRateFromMessage = regexp.MustCompile(`Pass rate:\s*([0-9]+(?:\.[0-9]+)?)%`)

// ParseQoSPassRateFromUpdateMessage extracts pass rate from engine-style incident update text.
func ParseQoSPassRateFromUpdateMessage(msg string) (float64, bool) {
	m := qosPassRateFromMessage.FindStringSubmatch(msg)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func scanIncident(row interface {
	Scan(dest ...any) error
}) (model.Incident, error) {
	var inc model.Incident
	var deg []byte
	err := row.Scan(&inc.ID, &inc.TargetType, &inc.TargetID, &inc.Title, &inc.Status, &inc.Severity,
		&inc.StartedAt, &inc.ResolvedAt, &inc.SourceTelemetryID, &deg, &inc.ResolvedBy, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		return inc, err
	}
	if len(deg) > 0 {
		inc.Degradation = json.RawMessage(deg)
	}
	return inc, nil
}

func (s *Store) ListIncidents(ctx context.Context, limit int, targetType string, since, until *time.Time) ([]model.Incident, error) {
	query := `SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at,
		source_telemetry_id, degradation, resolved_by, created_at, updated_at
	          FROM incidents`
	args := []any{}
	argIdx := 1
	conds := make([]string, 0, 3)

	if targetType != "" {
		conds = append(conds, fmt.Sprintf(`target_type = $%d`, argIdx))
		args = append(args, targetType)
		argIdx++
	}
	if since != nil {
		conds = append(conds, fmt.Sprintf(`started_at >= $%d`, argIdx))
		args = append(args, *since)
		argIdx++
	}
	if until != nil {
		conds = append(conds, fmt.Sprintf(`started_at < $%d`, argIdx))
		args = append(args, *until)
		argIdx++
	}
	if len(conds) > 0 {
		query += ` WHERE ` + conds[0]
		for i := 1; i < len(conds); i++ {
			query += ` AND ` + conds[i]
		}
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

	incidents := make([]model.Incident, 0)
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

// ListIncidentsForPublicTargets returns incidents overlapping the window [since, now] for the given
// public service and environment IDs (one query, capped). Overlap: started_at <= now AND
// (resolved_at IS NULL OR resolved_at >= since).
func (s *Store) ListIncidentsForPublicTargets(ctx context.Context, serviceIDs, environmentIDs []uuid.UUID, since time.Time) ([]model.Incident, error) {
	if len(serviceIDs) == 0 && len(environmentIDs) == 0 {
		return nil, nil
	}
	const limit = 500
	rows, err := s.pool.Query(ctx,
		`SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at,
			source_telemetry_id, degradation, resolved_by, created_at, updated_at
		 FROM incidents
		 WHERE started_at <= NOW()
		   AND (resolved_at IS NULL OR resolved_at >= $3)
		   AND (
		     (cardinality($1::uuid[]) > 0 AND target_type = 'service' AND target_id = ANY($1))
		     OR (cardinality($2::uuid[]) > 0 AND target_type = 'environment' AND target_id = ANY($2))
		   )
		 ORDER BY started_at DESC
		 LIMIT $4`,
		serviceIDs, environmentIDs, since, limit)
	if err != nil {
		return nil, fmt.Errorf("list incidents for public targets: %w", err)
	}
	defer rows.Close()

	var out []model.Incident
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		out = append(out, inc)
	}
	return out, rows.Err()
}

// IncidentUpdateBookends holds the chronologically first and last updates per incident.
type IncidentUpdateBookends struct {
	First *model.IncidentUpdate
	Last  *model.IncidentUpdate
}

// ListIncidentUpdateBookends loads the earliest and latest update row per incident in two queries.
func (s *Store) ListIncidentUpdateBookends(ctx context.Context, incidentIDs []uuid.UUID) (map[uuid.UUID]IncidentUpdateBookends, error) {
	out := make(map[uuid.UUID]IncidentUpdateBookends)
	if len(incidentIDs) == 0 {
		return out, nil
	}

	rowsFirst, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (incident_id) id, incident_id, status, message, author, created_at
		 FROM incident_updates WHERE incident_id = ANY($1::uuid[])
		 ORDER BY incident_id, created_at ASC`, incidentIDs)
	if err != nil {
		return nil, fmt.Errorf("list incident update first: %w", err)
	}
	defer rowsFirst.Close()
	for rowsFirst.Next() {
		var u model.IncidentUpdate
		if err := rowsFirst.Scan(&u.ID, &u.IncidentID, &u.Status, &u.Message, &u.Author, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan incident update first: %w", err)
		}
		b := out[u.IncidentID]
		uCopy := u
		b.First = &uCopy
		out[u.IncidentID] = b
	}
	if err := rowsFirst.Err(); err != nil {
		return nil, err
	}

	rowsLast, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (incident_id) id, incident_id, status, message, author, created_at
		 FROM incident_updates WHERE incident_id = ANY($1::uuid[])
		 ORDER BY incident_id, created_at DESC`, incidentIDs)
	if err != nil {
		return nil, fmt.Errorf("list incident update last: %w", err)
	}
	defer rowsLast.Close()
	for rowsLast.Next() {
		var u model.IncidentUpdate
		if err := rowsLast.Scan(&u.ID, &u.IncidentID, &u.Status, &u.Message, &u.Author, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan incident update last: %w", err)
		}
		b := out[u.IncidentID]
		uCopy := u
		b.Last = &uCopy
		out[u.IncidentID] = b
	}
	if err := rowsLast.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) GetIncidentByID(ctx context.Context, id uuid.UUID) (*model.Incident, error) {
	inc, err := scanIncident(s.pool.QueryRow(ctx,
		`SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at,
			source_telemetry_id, degradation, resolved_by, created_at, updated_at
		 FROM incidents WHERE id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}
	return &inc, nil
}

func (s *Store) GetOpenIncident(ctx context.Context, targetType string, targetID uuid.UUID) (*model.Incident, error) {
	inc, err := scanIncident(s.pool.QueryRow(ctx,
		`SELECT id, target_type, target_id, title, status, severity, started_at, resolved_at,
			source_telemetry_id, degradation, resolved_by, created_at, updated_at
		 FROM incidents WHERE target_type=$1 AND target_id=$2 AND resolved_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`, targetType, targetID))
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

func (s *Store) CreateIncident(ctx context.Context, targetType string, targetID uuid.UUID, title, severity string, sourceTelemetryID *uuid.UUID, degradation json.RawMessage) (*model.Incident, error) {
	if len(degradation) == 0 {
		degradation = json.RawMessage(`{}`)
	}
	var inc model.Incident
	deg := []byte(degradation)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO incidents (target_type, target_id, title, status, severity, source_telemetry_id, degradation)
		 VALUES ($1, $2, $3, 'investigating', $4, $5, $6::jsonb)
		 RETURNING id, target_type, target_id, title, status, severity, started_at, resolved_at,
			source_telemetry_id, degradation, resolved_by, created_at, updated_at`,
		targetType, targetID, title, severity, sourceTelemetryID, deg).
		Scan(&inc.ID, &inc.TargetType, &inc.TargetID, &inc.Title, &inc.Status, &inc.Severity,
			&inc.StartedAt, &inc.ResolvedAt, &inc.SourceTelemetryID, &deg, &inc.ResolvedBy, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create incident: %w", err)
	}
	if len(deg) > 0 {
		inc.Degradation = json.RawMessage(deg)
	}
	return &inc, nil
}

// UpdateIncidentDegradationAndSource refreshes probe context on an open incident.
func (s *Store) UpdateIncidentDegradationAndSource(ctx context.Context, id uuid.UUID, sourceTelemetryID *uuid.UUID, degradation json.RawMessage) error {
	if len(degradation) == 0 {
		degradation = json.RawMessage(`{}`)
	}
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE incidents SET source_telemetry_id = COALESCE($2, source_telemetry_id), degradation = $3::jsonb, updated_at = $4 WHERE id = $1`,
		id, sourceTelemetryID, []byte(degradation), now)
	if err != nil {
		return fmt.Errorf("update incident degradation: %w", err)
	}
	return nil
}

// UpdateIncidentStatus sets status. When status is resolved, pass resolvedBy as "system" or "admin"; otherwise nil.
func (s *Store) UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string, resolvedBy *string) error {
	now := time.Now()
	var resolvedAt *time.Time
	var rb *string
	if status == "resolved" {
		t := now
		resolvedAt = &t
		rb = resolvedBy
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE incidents SET status=$1, resolved_at=$2, resolved_by=$3, updated_at=$4 WHERE id=$5`,
		status, resolvedAt, rb, now, id)
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

	updates := make([]model.IncidentUpdate, 0)
	for rows.Next() {
		var u model.IncidentUpdate
		if err := rows.Scan(&u.ID, &u.IncidentID, &u.Status, &u.Message, &u.Author, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan incident update: %w", err)
		}
		updates = append(updates, u)
	}
	return updates, nil
}

// ListLatestQoSPassRateHints parses pass rate from the newest matching system update per incident (for rows without degradation JSON).
func (s *Store) ListLatestQoSPassRateHints(ctx context.Context, incidentIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	out := make(map[uuid.UUID]float64)
	if len(incidentIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (iu.incident_id) iu.incident_id, iu.message
		 FROM incident_updates iu
		 WHERE iu.incident_id = ANY($1::uuid[])
		   AND iu.author = 'system'
		   AND iu.message LIKE '%Pass rate%'
		 ORDER BY iu.incident_id, iu.created_at DESC`, incidentIDs)
	if err != nil {
		return nil, fmt.Errorf("list qos pass rate hints: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var msg string
		if err := rows.Scan(&id, &msg); err != nil {
			return nil, err
		}
		v, ok := ParseQoSPassRateFromUpdateMessage(msg)
		if !ok {
			continue
		}
		out[id] = v
	}
	return out, rows.Err()
}
