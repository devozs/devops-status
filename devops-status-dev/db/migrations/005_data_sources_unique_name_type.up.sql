-- Unique display identity per type: same name allowed for operational vs qos, not two of the same type.
CREATE UNIQUE INDEX IF NOT EXISTS telemetry_name_ds_type_uidx ON telemetry (name, ds_type);
