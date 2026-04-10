-- Case-insensitive unique telemetry names (Google vs google).
DROP INDEX IF EXISTS telemetry_name_uidx;
CREATE UNIQUE INDEX IF NOT EXISTS telemetry_name_lower_uidx ON telemetry (LOWER(name));
