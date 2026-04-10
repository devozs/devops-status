-- HL design changes 1: service providers, unified telemetry (no ds_type), env primary cluster,
-- telemetry links replacing probe bindings. No backward compatibility.
-- Idempotent: replaying must not delete telemetry/samples on every dev restart.

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'telemetry' AND column_name = 'ds_type'
  ) THEN
    DELETE FROM sample_results;
  END IF;
END $$;

DROP TABLE IF EXISTS service_probe_bindings;
DROP TABLE IF EXISTS environment_probe_bindings;

CREATE TABLE IF NOT EXISTS service_providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL DEFAULT '',
    host        VARCHAR(500) NOT NULL,
    port        INT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE services
    ADD COLUMN IF NOT EXISTS service_provider_id UUID REFERENCES service_providers(id) ON DELETE SET NULL;

ALTER TABLE environments
    ADD COLUMN IF NOT EXISTS k8s_cluster_id UUID REFERENCES k8s_clusters(id) ON DELETE SET NULL;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'telemetry' AND column_name = 'ds_type'
  ) THEN
    DELETE FROM secrets WHERE owner_type = 'telemetry';
    DELETE FROM telemetry;
  END IF;
END $$;

DROP INDEX IF EXISTS telemetry_name_ds_type_uidx;
ALTER TABLE telemetry DROP COLUMN IF EXISTS ds_type;

CREATE UNIQUE INDEX IF NOT EXISTS telemetry_name_uidx ON telemetry (name);

CREATE TABLE IF NOT EXISTS environment_telemetry_links (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id                  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    telemetry_id                    UUID NOT NULL REFERENCES telemetry(id) ON DELETE CASCADE,
    sample_interval_sec             INT NOT NULL DEFAULT 60,
    window_size                     INT NOT NULL DEFAULT 5,
    consecutive_failures_to_down    INT NOT NULL DEFAULT 3,
    consecutive_success_to_recover  INT NOT NULL DEFAULT 2,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (environment_id, telemetry_id)
);

CREATE TABLE IF NOT EXISTS service_telemetry_links (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id                      UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    telemetry_id                    UUID NOT NULL REFERENCES telemetry(id) ON DELETE CASCADE,
    sample_interval_sec             INT NOT NULL DEFAULT 60,
    window_size                     INT NOT NULL DEFAULT 5,
    consecutive_failures_to_down    INT NOT NULL DEFAULT 3,
    consecutive_success_to_recover  INT NOT NULL DEFAULT 2,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (service_id, telemetry_id)
);

CREATE INDEX IF NOT EXISTS idx_env_telemetry_links_env ON environment_telemetry_links (environment_id);
CREATE INDEX IF NOT EXISTS idx_svc_telemetry_links_svc ON service_telemetry_links (service_id);
