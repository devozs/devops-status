DROP INDEX IF EXISTS idx_svc_telemetry_links_svc;
DROP INDEX IF EXISTS idx_env_telemetry_links_env;

DROP TABLE IF EXISTS service_telemetry_links;
DROP TABLE IF EXISTS environment_telemetry_links;

DROP INDEX IF EXISTS telemetry_name_uidx;

ALTER TABLE telemetry
    ADD COLUMN IF NOT EXISTS ds_type VARCHAR(20) NOT NULL DEFAULT 'operational'
        CHECK (ds_type IN ('operational', 'qos'));

CREATE UNIQUE INDEX IF NOT EXISTS telemetry_name_ds_type_uidx ON telemetry (name, ds_type);

ALTER TABLE environments DROP COLUMN IF EXISTS k8s_cluster_id;

ALTER TABLE services DROP COLUMN IF EXISTS service_provider_id;

DROP TABLE IF EXISTS service_providers;

CREATE TABLE IF NOT EXISTS service_probe_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id      UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    probe_kind      VARCHAR(20) NOT NULL CHECK (probe_kind IN ('operational', 'qos')),
    telemetry_id    UUID NOT NULL REFERENCES telemetry(id) ON DELETE CASCADE,
    sample_interval_sec INT NOT NULL DEFAULT 60,
    window_size     INT NOT NULL DEFAULT 5,
    consecutive_failures_to_down INT NOT NULL DEFAULT 3,
    consecutive_success_to_recover INT NOT NULL DEFAULT 2,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS environment_probe_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    probe_kind      VARCHAR(20) NOT NULL CHECK (probe_kind IN ('operational', 'qos')),
    telemetry_id    UUID NOT NULL REFERENCES telemetry(id) ON DELETE CASCADE,
    sample_interval_sec INT NOT NULL DEFAULT 60,
    window_size     INT NOT NULL DEFAULT 5,
    consecutive_failures_to_down INT NOT NULL DEFAULT 3,
    consecutive_success_to_recover INT NOT NULL DEFAULT 2,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
