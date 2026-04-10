-- Last result of admin "Verify API" (GET /version from backend to cluster). Separate from handshake status.
ALTER TABLE k8s_clusters ADD COLUMN IF NOT EXISTS api_verify_ok BOOLEAN;
ALTER TABLE k8s_clusters ADD COLUMN IF NOT EXISTS api_verify_checked_at TIMESTAMPTZ;
ALTER TABLE k8s_clusters ADD COLUMN IF NOT EXISTS api_verify_detail TEXT;
