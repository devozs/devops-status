-- Restore adapter check without liveness (fails if any telemetry row uses adapter = liveness).
ALTER TABLE telemetry DROP CONSTRAINT IF EXISTS telemetry_adapter_check;

ALTER TABLE telemetry
    ADD CONSTRAINT telemetry_adapter_check
        CHECK (adapter IN ('prometheus', 'http', 'cli', 'kubernetes'));
