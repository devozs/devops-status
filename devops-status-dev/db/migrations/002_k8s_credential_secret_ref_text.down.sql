-- Revert is lossy if any secret_ref > 500 chars; avoid running down in prod with long tokens.
ALTER TABLE k8s_cluster_credentials
  ALTER COLUMN secret_ref TYPE VARCHAR(500);
