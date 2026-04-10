DROP INDEX IF EXISTS idx_incidents_source_telemetry;

ALTER TABLE incidents
    DROP COLUMN IF EXISTS resolved_by,
    DROP COLUMN IF EXISTS degradation,
    DROP COLUMN IF EXISTS source_telemetry_id;
