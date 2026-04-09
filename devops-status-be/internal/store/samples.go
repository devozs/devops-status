package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SampleRecord struct {
	ID           uuid.UUID      `json:"id"`
	TargetType   string         `json:"target_type"`
	TargetID     uuid.UUID      `json:"target_id"`
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
		`INSERT INTO sample_results (target_type, target_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rec.TargetType, rec.TargetID, rec.ProbeKind, rec.SampledAt, rec.Success, rec.RawValue, rec.LatencyMs, metaBytes, rec.SourceTrace)
	if err != nil {
		return fmt.Errorf("insert sample: %w", err)
	}
	return nil
}

func (s *Store) GetRecentSamples(ctx context.Context, targetType string, targetID uuid.UUID, probeKind string, limit int) ([]SampleRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, target_type, target_id, probe_kind, sampled_at, success, raw_value, latency_ms, metadata_json, source_trace
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
		var rec SampleRecord
		var metaBytes []byte
		if err := rows.Scan(&rec.ID, &rec.TargetType, &rec.TargetID, &rec.ProbeKind, &rec.SampledAt,
			&rec.Success, &rec.RawValue, &rec.LatencyMs, &metaBytes, &rec.SourceTrace); err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		json.Unmarshal(metaBytes, &rec.MetadataJSON)
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

func (s *Store) GetDailyRollups(ctx context.Context, targetType string, targetID uuid.UUID, days int) ([]DailyRollup, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT date_trunc('day', sampled_at)::date AS day,
		        COUNT(*)::int AS total,
		        COUNT(*) FILTER (WHERE NOT success)::int AS failed,
		        CASE WHEN COUNT(*) > 0 THEN ROUND(100.0 * COUNT(*) FILTER (WHERE success) / COUNT(*), 2) ELSE 100 END AS avail_pct
		 FROM sample_results
		 WHERE target_type=$1 AND target_id=$2 AND sampled_at >= now() - ($3 || ' days')::interval
		 GROUP BY day ORDER BY day ASC`,
		targetType, targetID, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, fmt.Errorf("get daily rollups: %w", err)
	}
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
	return rollups, nil
}
