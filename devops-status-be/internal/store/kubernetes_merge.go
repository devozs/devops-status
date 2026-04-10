package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MergeKubernetesProbeConfig fills token, endpoint, and namespace from registered cluster credentials
// when config contains cluster_id and those fields are empty.
// Applies to adapter kubernetes, and to adapter cli when execution_target is k8s_cluster.
// If defaultInsecureSkipTLS is true and the config does not set insecure_skip_tls, it is set to true (private CA / dev clusters).
func (s *Store) MergeKubernetesProbeConfig(ctx context.Context, adapter string, raw json.RawMessage, defaultInsecureSkipTLS bool) (json.RawMessage, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return raw, err
	}
	if adapter == "cli" {
		if et, _ := cfg["execution_target"].(string); et != "k8s_cluster" {
			return raw, nil
		}
	} else if adapter != "kubernetes" {
		return raw, nil
	}
	cid, _ := cfg["cluster_id"].(string)
	if cid == "" {
		return raw, nil
	}
	clusterUUID, err := uuid.Parse(cid)
	if err != nil {
		return raw, nil
	}

	_, token, err := s.GetActiveClusterCredential(ctx, clusterUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return raw, nil
		}
		return nil, err
	}
	if token != "" {
		if existing, _ := cfg["token"].(string); existing == "" {
			cfg["token"] = token
		}
	}

	cluster, err := s.GetK8sClusterByID(ctx, clusterUUID)
	if err != nil {
		return nil, err
	}
	if ep, _ := cfg["endpoint"].(string); ep == "" && cluster.Endpoint != "" {
		cfg["endpoint"] = cluster.Endpoint
	}
	if ns, _ := cfg["namespace"].(string); ns == "" && cluster.DefaultNamespace != "" {
		cfg["namespace"] = cluster.DefaultNamespace
	}

	if defaultInsecureSkipTLS {
		if _, exists := cfg["insecure_skip_tls"]; !exists {
			cfg["insecure_skip_tls"] = true
		}
	}

	out, err := json.Marshal(cfg)
	if err != nil {
		return raw, err
	}
	return out, nil
}
