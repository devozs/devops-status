-- Admin users (Phase 0 local auth)
CREATE TABLE IF NOT EXISTS admin_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role        VARCHAR(50) NOT NULL DEFAULT 'admin',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    last_login  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Environments
CREATE TABLE IF NOT EXISTS environments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    env_type    VARCHAR(50) NOT NULL DEFAULT 'custom',
    is_public   BOOLEAN NOT NULL DEFAULT true,
    criticality VARCHAR(50) NOT NULL DEFAULT 'standard',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Services
CREATE TABLE IF NOT EXISTS services (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_public   BOOLEAN NOT NULL DEFAULT true,
    criticality VARCHAR(50) NOT NULL DEFAULT 'standard',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Environment-service membership (many-to-many)
CREATE TABLE IF NOT EXISTS environment_service_membership (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id     UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(environment_id, service_id)
);

-- Data sources
CREATE TABLE IF NOT EXISTS data_sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    ds_type     VARCHAR(20) NOT NULL CHECK (ds_type IN ('operational', 'qos')),
    adapter     VARCHAR(20) NOT NULL CHECK (adapter IN ('prometheus', 'http', 'cli', 'kubernetes')),
    config_json JSONB NOT NULL DEFAULT '{}',
    secret_ref  VARCHAR(500) NOT NULL DEFAULT '',
    timeout_ms  INT NOT NULL DEFAULT 30000,
    retries     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Kubernetes clusters
CREATE TABLE IF NOT EXISTS k8s_clusters (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,
    environment_id   UUID REFERENCES environments(id) ON DELETE SET NULL,
    endpoint         VARCHAR(500) NOT NULL,
    auth_method      VARCHAR(50) NOT NULL DEFAULT 'service_account_token',
    credential_ref   VARCHAR(500) NOT NULL DEFAULT '',
    default_namespace VARCHAR(255) NOT NULL DEFAULT 'default',
    namespace_filter  JSONB NOT NULL DEFAULT '[]',
    k8s_version      VARCHAR(50) NOT NULL DEFAULT '',
    status           VARCHAR(50) NOT NULL DEFAULT 'pending_registration',
    last_capability_scan TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Kubernetes cluster handshakes
CREATE TABLE IF NOT EXISTS k8s_cluster_handshakes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id      UUID NOT NULL REFERENCES k8s_clusters(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    registered_at   TIMESTAMPTZ,
    last_heartbeat  TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    metadata_json   JSONB NOT NULL DEFAULT '{}'
);

-- Kubernetes cluster credentials (secure refs only)
CREATE TABLE IF NOT EXISTS k8s_cluster_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id      UUID NOT NULL REFERENCES k8s_clusters(id) ON DELETE CASCADE,
    credential_type VARCHAR(50) NOT NULL,
    secret_ref      VARCHAR(500) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ
);

-- Kubernetes API capabilities (per cluster)
CREATE TABLE IF NOT EXISTS k8s_api_capabilities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id      UUID NOT NULL REFERENCES k8s_clusters(id) ON DELETE CASCADE,
    api_group       VARCHAR(255) NOT NULL,
    api_version     VARCHAR(100) NOT NULL,
    resource        VARCHAR(255) NOT NULL,
    is_preferred    BOOLEAN NOT NULL DEFAULT false,
    is_deprecated   BOOLEAN NOT NULL DEFAULT false,
    discovered_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Kubernetes probe templates (version-independent)
CREATE TABLE IF NOT EXISTS k8s_probe_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    resource_intent VARCHAR(100) NOT NULL,
    probe_type      VARCHAR(20) NOT NULL CHECK (probe_type IN ('operational', 'qos')),
    config_json     JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Kubernetes probe resolutions (cluster-specific GVR mapping)
CREATE TABLE IF NOT EXISTS k8s_probe_resolutions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id     UUID NOT NULL REFERENCES k8s_probe_templates(id) ON DELETE CASCADE,
    cluster_id      UUID NOT NULL REFERENCES k8s_clusters(id) ON DELETE CASCADE,
    resolved_group  VARCHAR(255) NOT NULL,
    resolved_version VARCHAR(100) NOT NULL,
    resolved_resource VARCHAR(255) NOT NULL,
    is_override     BOOLEAN NOT NULL DEFAULT false,
    resolved_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(template_id, cluster_id)
);

-- Service probe bindings
CREATE TABLE IF NOT EXISTS service_probe_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id      UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    probe_kind      VARCHAR(20) NOT NULL CHECK (probe_kind IN ('operational', 'qos')),
    data_source_id  UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    sample_interval_sec INT NOT NULL DEFAULT 60,
    window_size     INT NOT NULL DEFAULT 5,
    consecutive_failures_to_down INT NOT NULL DEFAULT 3,
    consecutive_success_to_recover INT NOT NULL DEFAULT 2,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Environment probe bindings
CREATE TABLE IF NOT EXISTS environment_probe_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    probe_kind      VARCHAR(20) NOT NULL CHECK (probe_kind IN ('operational', 'qos')),
    data_source_id  UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    sample_interval_sec INT NOT NULL DEFAULT 60,
    window_size     INT NOT NULL DEFAULT 5,
    consecutive_failures_to_down INT NOT NULL DEFAULT 3,
    consecutive_success_to_recover INT NOT NULL DEFAULT 2,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- QoS policies
CREATE TABLE IF NOT EXISTS qos_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type     VARCHAR(20) NOT NULL CHECK (target_type IN ('service', 'environment')),
    target_id       UUID NOT NULL,
    green_threshold NUMERIC(5,2) NOT NULL DEFAULT 99.0,
    yellow_threshold NUMERIC(5,2) NOT NULL DEFAULT 95.0,
    metric_type     VARCHAR(50) NOT NULL DEFAULT 'pass_rate',
    config_json     JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sample results (partitioned monthly later; flat for Phase 0)
CREATE TABLE IF NOT EXISTS sample_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type     VARCHAR(20) NOT NULL CHECK (target_type IN ('service', 'environment')),
    target_id       UUID NOT NULL,
    probe_kind      VARCHAR(20) NOT NULL CHECK (probe_kind IN ('operational', 'qos')),
    sampled_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    success         BOOLEAN NOT NULL,
    raw_value       NUMERIC,
    latency_ms      INT,
    metadata_json   JSONB NOT NULL DEFAULT '{}',
    source_trace    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sample_results_target ON sample_results(target_type, target_id, sampled_at DESC);

-- Status rollups (bucketed snapshots for fast reads)
CREATE TABLE IF NOT EXISTS status_rollups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type     VARCHAR(20) NOT NULL CHECK (target_type IN ('service', 'environment')),
    target_id       UUID NOT NULL,
    bucket_start    TIMESTAMPTZ NOT NULL,
    bucket_size     VARCHAR(20) NOT NULL DEFAULT 'hourly',
    availability_pct NUMERIC(5,2),
    qos_level       VARCHAR(10) CHECK (qos_level IN ('green', 'yellow', 'red')),
    total_samples   INT NOT NULL DEFAULT 0,
    failed_samples  INT NOT NULL DEFAULT 0,
    avg_latency_ms  NUMERIC,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_status_rollups_bucket ON status_rollups(target_type, target_id, bucket_start, bucket_size);

-- Incidents
CREATE TABLE IF NOT EXISTS incidents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type     VARCHAR(20) NOT NULL CHECK (target_type IN ('service', 'environment')),
    target_id       UUID NOT NULL,
    title           VARCHAR(500) NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'investigating',
    severity        VARCHAR(50) NOT NULL DEFAULT 'minor',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_incidents_target ON incidents(target_type, target_id, started_at DESC);

-- Incident updates
CREATE TABLE IF NOT EXISTS incident_updates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id     UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    status          VARCHAR(50) NOT NULL,
    message         TEXT NOT NULL,
    author          VARCHAR(255) NOT NULL DEFAULT 'system',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel         VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'teams')),
    target          VARCHAR(500) NOT NULL,
    scope           VARCHAR(50) NOT NULL DEFAULT 'all',
    is_verified     BOOLEAN NOT NULL DEFAULT false,
    verify_token    VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    unsubscribed_at TIMESTAMPTZ
);

-- Notification events (audit trail)
CREATE TABLE IF NOT EXISTS notification_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    incident_id     UUID REFERENCES incidents(id) ON DELETE SET NULL,
    channel         VARCHAR(20) NOT NULL,
    payload_json    JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts        INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sessions table for server-side session storage
CREATE TABLE IF NOT EXISTS sessions (
    id          VARCHAR(255) PRIMARY KEY,
    data        BYTEA NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Audit log for admin actions
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id   UUID,
    details_json JSONB NOT NULL DEFAULT '{}',
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);
