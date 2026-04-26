CREATE TABLE IF NOT EXISTS telemetry_shell_hints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind VARCHAR(32) NOT NULL CHECK (kind IN ('job_env', 'job_node_selector', 'container_prep', 'probe_shell')),
    title VARCHAR(256) NOT NULL,
    body TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kind, title)
);

CREATE INDEX IF NOT EXISTS idx_telemetry_shell_hints_kind_order ON telemetry_shell_hints (kind, sort_order, title);
