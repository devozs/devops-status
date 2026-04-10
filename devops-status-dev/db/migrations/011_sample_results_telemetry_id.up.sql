ALTER TABLE sample_results
    ADD COLUMN IF NOT EXISTS telemetry_id UUID REFERENCES telemetry(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sample_results_target_telemetry_time
    ON sample_results (target_type, target_id, telemetry_id, sampled_at DESC);
