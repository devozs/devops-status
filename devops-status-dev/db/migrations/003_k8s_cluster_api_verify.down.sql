ALTER TABLE k8s_clusters DROP COLUMN IF EXISTS api_verify_detail;
ALTER TABLE k8s_clusters DROP COLUMN IF EXISTS api_verify_checked_at;
ALTER TABLE k8s_clusters DROP COLUMN IF EXISTS api_verify_ok;
