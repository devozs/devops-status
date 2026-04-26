ALTER TABLE service_providers DROP CONSTRAINT IF EXISTS service_providers_provider_type_check;

ALTER TABLE service_providers
    ADD CONSTRAINT service_providers_provider_type_check
        CHECK (provider_type IN (
            'tcp',
            'prometheus',
            'grafana',
            'elasticsearch',
            'jenkins',
            'artifactory',
            'dns',
            'rancher'
        ));

DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN (
    SELECT c.conname
    FROM pg_constraint c
    WHERE c.conrelid = 'telemetry'::regclass
      AND c.contype = 'c'
      AND pg_get_constraintdef(c.oid) LIKE '%adapter%'
  ) LOOP
    EXECUTE format('ALTER TABLE telemetry DROP CONSTRAINT %I', r.conname);
  END LOOP;
END $$;

ALTER TABLE telemetry
    ADD CONSTRAINT telemetry_adapter_check
        CHECK (adapter IN ('prometheus', 'http', 'cli', 'kubernetes', 'liveness'));
