package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SampleRecord struct {
	ID           uuid.UUID      `json:"id"`
	TargetType   string         `json:"target_type"`
	TargetID     uuid.UUID      `json:"target_id"`
	TelemetryID  *uuid.UUID     `json:"telemetry_id,omitempty"`
	ProbeKind    string         `json:"probe_kind"`
	SampledAt    time.Time      `json:"sampled_at"`
	Success      bool           `json:"success"`
	RawValue     *float64       `json:"raw_value,omitempty"`
	LatencyMs    *int           `json:"latency_ms,omitempty"`
	MetadataJSON map[string]any `json:"metadata_json,omitempty"`
	SourceTrace  string         `json:"source_trace,omitempty"`
}

func (s *Store) InsertSampleResult(ctx context.Context, rec SampleRecord) error {
	metaBytes, _ := json.Marshal(rec.MetadataJSON)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sample_results (target_type, target_id, telemetry_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rec.TargetType, rec.TargetID, rec.TelemetryID, rec.ProbeKind, rec.SampledAt, rec.Success, rec.RawValue, rec.LatencyMs, metaBytes, rec.SourceTrace)
	if err != nil {
		return fmt.Errorf("insert sample: %w", err)
	}
	return nil
}

func scanSampleRow(rows interface {
	Scan(dest ...any) error
}) (SampleRecord, error) {
	var rec SampleRecord
	var metaBytes []byte
	var tid *uuid.UUID
	if err := rows.Scan(&rec.ID, &rec.TargetType, &rec.TargetID, &tid, &rec.ProbeKind, &rec.SampledAt,
		&rec.Success, &rec.RawValue, &rec.LatencyMs, &metaBytes, &rec.SourceTrace); err != nil {
		return rec, err
	}
	rec.TelemetryID = tid
	_ = json.Unmarshal(metaBytes, &rec.MetadataJSON)
	return rec, nil
}

func (s *Store) GetRecentSamples(ctx context.Context, targetType string, targetID uuid.UUID, probeKind string, limit int) ([]SampleRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, target_type, target_id, telemetry_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace
		 FROM sample_results
		 WHERE target_type=$1 AND target_id=$2 AND probe_kind=$3
		 ORDER BY sampled_at DESC LIMIT $4`,
		targetType, targetID, probeKind, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent samples: %w", err)
	}
	defer rows.Close()

	var samples []SampleRecord
	for rows.Next() {
		rec, err := scanSampleRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		samples = append(samples, rec)
	}
	return samples, nil
}

// GetRecentSamplesForProbe returns recent samples for one probe stream. When telemetryID is non-nil, only rows with that
// telemetry_id are returned; when nil, only rows with telemetry_id IS NULL (legacy aggregate stream).
func (s *Store) GetRecentSamplesForProbe(ctx context.Context, targetType string, targetID uuid.UUID, telemetryID *uuid.UUID, probeKind string, limit int) ([]SampleRecord, error) {
	var rows pgx.Rows
	var err error
	if telemetryID != nil {
		rows, err = s.pool.Query(ctx,
			`SELECT id, target_type, target_id, telemetry_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace
			 FROM sample_results
			 WHERE target_type=$1 AND target_id=$2 AND probe_kind=$3 AND telemetry_id=$4
			 ORDER BY sampled_at DESC LIMIT $5`,
			targetType, targetID, probeKind, *telemetryID, limit)
	} else {
		rows, err = s.pool.Query(ctx,
			`SELECT id, target_type, target_id, telemetry_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace
			 FROM sample_results
			 WHERE target_type=$1 AND target_id=$2 AND probe_kind=$3 AND telemetry_id IS NULL
			 ORDER BY sampled_at DESC LIMIT $4`,
			targetType, targetID, probeKind, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("get recent samples for probe: %w", err)
	}
	defer rows.Close()

	var samples []SampleRecord
	for rows.Next() {
		rec, err := scanSampleRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		samples = append(samples, rec)
	}
	return samples, nil
}

// ListRecentSamplesForEnvironmentTelemetry returns recent samples for one environment telemetry link (telemetry_id must match).
func (s *Store) ListRecentSamplesForEnvironmentTelemetry(ctx context.Context, envID, telemetryID uuid.UUID, probeKind string, limit int) ([]SampleRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, target_type, target_id, telemetry_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace
		 FROM sample_results
		 WHERE target_type='environment' AND target_id=$1 AND telemetry_id=$2 AND probe_kind=$3
		 ORDER BY sampled_at DESC LIMIT $4`,
		envID, telemetryID, probeKind, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent samples for environment telemetry: %w", err)
	}
	defer rows.Close()

	var samples []SampleRecord
	for rows.Next() {
		rec, err := scanSampleRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		samples = append(samples, rec)
	}
	return samples, nil
}

// ListRecentSamplesForServiceTelemetry returns recent samples for one service telemetry link (telemetry_id must match).
func (s *Store) ListRecentSamplesForServiceTelemetry(ctx context.Context, serviceID, telemetryID uuid.UUID, probeKind string, limit int) ([]SampleRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, target_type, target_id, telemetry_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace
		 FROM sample_results
		 WHERE target_type='service' AND target_id=$1 AND telemetry_id=$2 AND probe_kind=$3
		 ORDER BY sampled_at DESC LIMIT $4`,
		serviceID, telemetryID, probeKind, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent samples for service telemetry: %w", err)
	}
	defer rows.Close()

	var samples []SampleRecord
	for rows.Next() {
		rec, err := scanSampleRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		samples = append(samples, rec)
	}
	return samples, nil
}

type DailyRollup struct {
	Date            string  `json:"date"`
	AvailabilityPct float64 `json:"availability_pct"`
	QoSLevel        string  `json:"qos_level"`
	TotalSamples    int     `json:"total_samples"`
	FailedSamples   int     `json:"failed_samples"`
}

func dailyRollupsFromRows(rows pgx.Rows) ([]DailyRollup, error) {
	defer rows.Close()

	var rollups []DailyRollup
	for rows.Next() {
		var r DailyRollup
		var day time.Time
		if err := rows.Scan(&day, &r.TotalSamples, &r.FailedSamples, &r.AvailabilityPct); err != nil {
			return nil, fmt.Errorf("scan rollup: %w", err)
		}
		r.Date = day.Format("2006-01-02")
		if r.AvailabilityPct >= 99 {
			r.QoSLevel = "green"
		} else if r.AvailabilityPct >= 95 {
			r.QoSLevel = "yellow"
		} else {
			r.QoSLevel = "red"
		}
		rollups = append(rollups, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rollup rows: %w", err)
	}
	return rollups, nil
}

// GetDailyRollups returns daily availability for [endExclusive - days, endExclusive) in UTC.
// If endExclusiveUTC is nil, end is start of tomorrow UTC (include all samples through today UTC).
func (s *Store) GetDailyRollups(ctx context.Context, targetType string, targetID uuid.UUID, days int, endExclusiveUTC *time.Time) ([]DailyRollup, error) {
	end := defaultRollupEndExclusive(endExclusiveUTC)
	rows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', sampled_at)::date AS day,
		        COUNT(*)::int AS total,
		        COUNT(*) FILTER (WHERE NOT success)::int AS failed,
		        CASE WHEN COUNT(*) > 0 THEN ROUND(100.0 * COUNT(*) FILTER (WHERE success) / COUNT(*), 2) ELSE 100 END AS avail_pct
		 FROM sample_results
		 WHERE target_type=$1 AND target_id=$2
		   AND sampled_at >= $4::timestamptz - ($3 || ' days')::interval
		   AND sampled_at < $4::timestamptz
		 GROUP BY day ORDER BY day ASC`,
		targetType, targetID, fmt.Sprintf("%d", days), end)
	if err != nil {
		return nil, fmt.Errorf("get daily rollups: %w", err)
	}
	return dailyRollupsFromRows(rows)
}

func defaultRollupEndExclusive(endExclusiveUTC *time.Time) time.Time {
	if endExclusiveUTC != nil && !endExclusiveUTC.IsZero() {
		return *endExclusiveUTC
	}
	now := time.Now().UTC()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
}

// GetDailyRollupsForEnvironmentTelemetry limits rollups to samples tagged with the given telemetry_id (per-link series).
func (s *Store) GetDailyRollupsForEnvironmentTelemetry(ctx context.Context, envID, telemetryID uuid.UUID, days int) ([]DailyRollup, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', sampled_at)::date AS day,
		        COUNT(*)::int AS total,
		        COUNT(*) FILTER (WHERE NOT success)::int AS failed,
		        CASE WHEN COUNT(*) > 0 THEN ROUND(100.0 * COUNT(*) FILTER (WHERE success) / COUNT(*), 2) ELSE 100 END AS avail_pct
		 FROM sample_results
		 WHERE target_type='environment' AND target_id=$1 AND telemetry_id=$2
		   AND sampled_at >= now() - ($3 || ' days')::interval
		 GROUP BY day ORDER BY day ASC`,
		envID, telemetryID, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, fmt.Errorf("get daily rollups for environment telemetry: %w", err)
	}
	return dailyRollupsFromRows(rows)
}

// GetDailyRollupsForServiceTelemetry limits rollups to samples tagged with the given telemetry_id (per-link series).
func (s *Store) GetDailyRollupsForServiceTelemetry(ctx context.Context, serviceID, telemetryID uuid.UUID, days int) ([]DailyRollup, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', sampled_at)::date AS day,
		        COUNT(*)::int AS total,
		        COUNT(*) FILTER (WHERE NOT success)::int AS failed,
		        CASE WHEN COUNT(*) > 0 THEN ROUND(100.0 * COUNT(*) FILTER (WHERE success) / COUNT(*), 2) ELSE 100 END AS avail_pct
		 FROM sample_results
		 WHERE target_type='service' AND target_id=$1 AND telemetry_id=$2
		   AND sampled_at >= now() - ($3 || ' days')::interval
		 GROUP BY day ORDER BY day ASC`,
		serviceID, telemetryID, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, fmt.Errorf("get daily rollups for service telemetry: %w", err)
	}
	return dailyRollupsFromRows(rows)
}
