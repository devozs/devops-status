-- Allow liveness adapter (unified telemetry); app already validates adapter in code.
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
