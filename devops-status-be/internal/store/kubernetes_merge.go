package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MergeKubernetesProbeConfig fills token, endpoint, and namespace from registered cluster credentials
// when config contains cluster_id and those fields are empty.
// Applies to adapter kubernetes, liveness (source=kubernetes nested), and to adapter cli when execution_target is k8s_cluster.
// If defaultInsecureSkipTLS is true and the config does not set insecure_skip_tls, it is set to true (private CA / dev clusters).
func (s *Store) MergeKubernetesProbeConfig(ctx context.Context, adapter string, raw json.RawMessage, defaultInsecureSkipTLS bool) (json.RawMessage, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return raw, err
	}

	switch adapter {
	case "cli":
		if et, _ := cfg["execution_target"].(string); et != "k8s_cluster" {
			return raw, nil
		}
		merged, err := s.mergeKubernetesConfigMap(ctx, cfg, defaultInsecureSkipTLS)
		if err != nil {
			return nil, err
		}
		out, err := json.Marshal(merged)
		if err != nil {
			return raw, err
		}
		return out, nil

	case "kubernetes":
		merged, err := s.mergeKubernetesConfigMap(ctx, cfg, defaultInsecureSkipTLS)
		if err != nil {
			return nil, err
		}
		out, err := json.Marshal(merged)
		if err != nil {
			return raw, err
		}
		return out, nil

	case "liveness":
		src, _ := cfg["source"].(string)
		if strings.ToLower(strings.TrimSpace(src)) != "kubernetes" {
			return raw, nil
		}
		nested, ok := cfg["kubernetes"].(map[string]any)
		if !ok || nested == nil {
			return raw, nil
		}
		mergedNest, err := s.mergeKubernetesConfigMap(ctx, nested, defaultInsecureSkipTLS)
		if err != nil {
			return nil, err
		}
		cfg["kubernetes"] = mergedNest
		out, err := json.Marshal(cfg)
		if err != nil {
			return raw, err
		}
		return out, nil

	default:
		return raw, nil
	}
}

// mergeKubernetesConfigMap expects top-level keys like cluster_id; mutates and returns the same map.
func (s *Store) mergeKubernetesConfigMap(ctx context.Context, cfg map[string]any, defaultInsecureSkipTLS bool) (map[string]any, error) {
	cid, _ := cfg["cluster_id"].(string)
	if cid == "" {
		return cfg, nil
	}
	clusterUUID, err := uuid.Parse(cid)
	if err != nil {
		return cfg, nil
	}

	_, token, err := s.GetActiveClusterCredential(ctx, clusterUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cfg, nil
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

	return cfg, nil
}
