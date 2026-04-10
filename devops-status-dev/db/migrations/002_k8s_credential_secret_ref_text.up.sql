-- Service account tokens exceed VARCHAR(500); store full credential material for dev (prod should use external secret refs).
ALTER TABLE k8s_cluster_credentials
  ALTER COLUMN secret_ref TYPE TEXT;
