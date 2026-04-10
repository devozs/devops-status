DROP INDEX IF EXISTS idx_sample_results_target_telemetry_time;

ALTER TABLE sample_results DROP COLUMN IF EXISTS telemetry_id;
