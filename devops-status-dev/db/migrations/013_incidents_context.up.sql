ALTER TABLE incidents
    ADD COLUMN IF NOT EXISTS source_telemetry_id UUID REFERENCES telemetry(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS degradation JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS resolved_by VARCHAR(20) CHECK (resolved_by IS NULL OR resolved_by IN ('system', 'admin'));

CREATE INDEX IF NOT EXISTS idx_incidents_source_telemetry ON incidents(source_telemetry_id);
