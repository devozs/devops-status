-- Telemetry hardening: clean slate for bindings + telemetry, new columns, secrets table.
-- Idempotent: replaying this migration must not wipe data. Only clean slate when the
-- telemetry row shape is still pre-hardening (qos_thresholds not present yet).
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'environment_probe_bindings'
  ) THEN
    DELETE FROM environment_probe_bindings;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'service_probe_bindings'
  ) THEN
    DELETE FROM service_probe_bindings;
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'telemetry' AND column_name = 'qos_thresholds'
  ) THEN
    DELETE FROM telemetry;
  END IF;
END $$;

ALTER TABLE telemetry
    ADD COLUMN IF NOT EXISTS qos_thresholds JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS execution_target VARCHAR(20) NOT NULL DEFAULT 'backend'
        CHECK (execution_target IN ('backend', 'k8s_cluster'));

CREATE TABLE IF NOT EXISTS secrets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_type  VARCHAR(50) NOT NULL,
    owner_id    UUID NOT NULL,
    key         VARCHAR(100) NOT NULL,
    cipher_text BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_type, owner_id, key)
);

CREATE INDEX IF NOT EXISTS secrets_owner_idx ON secrets (owner_type, owner_id);
