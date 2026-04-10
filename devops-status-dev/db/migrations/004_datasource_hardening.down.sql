DROP INDEX IF EXISTS secrets_owner_idx;
DROP TABLE IF EXISTS secrets;

ALTER TABLE telemetry DROP COLUMN IF EXISTS execution_target;
ALTER TABLE telemetry DROP COLUMN IF EXISTS qos_thresholds;
