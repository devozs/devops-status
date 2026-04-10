DROP INDEX IF EXISTS telemetry_name_lower_uidx;
CREATE UNIQUE INDEX IF NOT EXISTS telemetry_name_uidx ON telemetry (name);
