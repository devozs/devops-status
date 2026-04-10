-- Service provider: TCP vs Prometheus (config in JSONB, credentials in secrets).
ALTER TABLE service_providers
    ADD COLUMN IF NOT EXISTS provider_type VARCHAR(32) NOT NULL DEFAULT 'tcp'
        CHECK (provider_type IN ('tcp', 'prometheus')),
    ADD COLUMN IF NOT EXISTS config_json JSONB NOT NULL DEFAULT '{}';

UPDATE service_providers SET provider_type = 'tcp', config_json = '{}' WHERE provider_type IS NULL OR config_json IS NULL;
