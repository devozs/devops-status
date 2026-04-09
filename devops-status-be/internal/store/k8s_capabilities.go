package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type K8sAPICapability struct {
	ID           uuid.UUID `json:"id"`
	ClusterID    uuid.UUID `json:"cluster_id"`
	APIGroup     string    `json:"api_group"`
	APIVersion   string    `json:"api_version"`
	Resource     string    `json:"resource"`
	IsPreferred  bool      `json:"is_preferred"`
	IsDeprecated bool      `json:"is_deprecated"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

func (s *Store) UpsertK8sCapabilities(ctx context.Context, clusterID uuid.UUID, caps []K8sAPICapability) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM k8s_api_capabilities WHERE cluster_id = $1`, clusterID)
	if err != nil {
		return fmt.Errorf("clear old capabilities: %w", err)
	}
	for _, c := range caps {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO k8s_api_capabilities (cluster_id, api_group, api_version, resource, is_preferred, is_deprecated)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			clusterID, c.APIGroup, c.APIVersion, c.Resource, c.IsPreferred, c.IsDeprecated)
		if err != nil {
			return fmt.Errorf("insert capability: %w", err)
		}
	}
	return nil
}

func (s *Store) ListK8sCapabilities(ctx context.Context, clusterID uuid.UUID) ([]K8sAPICapability, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, cluster_id, api_group, api_version, resource, is_preferred, is_deprecated, discovered_at
		 FROM k8s_api_capabilities WHERE cluster_id=$1 ORDER BY api_group, resource, api_version`, clusterID)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	defer rows.Close()

	var caps []K8sAPICapability
	for rows.Next() {
		var c K8sAPICapability
		if err := rows.Scan(&c.ID, &c.ClusterID, &c.APIGroup, &c.APIVersion, &c.Resource, &c.IsPreferred, &c.IsDeprecated, &c.DiscoveredAt); err != nil {
			return nil, fmt.Errorf("scan capability: %w", err)
		}
		caps = append(caps, c)
	}
	return caps, nil
}

func (s *Store) FindPreferredAPIVersion(ctx context.Context, clusterID uuid.UUID, resource string) (*K8sAPICapability, error) {
	var c K8sAPICapability
	err := s.pool.QueryRow(ctx,
		`SELECT id, cluster_id, api_group, api_version, resource, is_preferred, is_deprecated, discovered_at
		 FROM k8s_api_capabilities
		 WHERE cluster_id=$1 AND resource=$2 AND is_deprecated=false
		 ORDER BY is_preferred DESC, api_version DESC
		 LIMIT 1`, clusterID, resource).
		Scan(&c.ID, &c.ClusterID, &c.APIGroup, &c.APIVersion, &c.Resource, &c.IsPreferred, &c.IsDeprecated, &c.DiscoveredAt)
	if err != nil {
		return nil, fmt.Errorf("find preferred version for %s: %w", resource, err)
	}
	return &c, nil
}

type K8sProbeResolution struct {
	ID               uuid.UUID `json:"id"`
	TemplateID       uuid.UUID `json:"template_id"`
	ClusterID        uuid.UUID `json:"cluster_id"`
	ResolvedGroup    string    `json:"resolved_group"`
	ResolvedVersion  string    `json:"resolved_version"`
	ResolvedResource string    `json:"resolved_resource"`
	IsOverride       bool      `json:"is_override"`
	ResolvedAt       time.Time `json:"resolved_at"`
}

func (s *Store) UpsertProbeResolution(ctx context.Context, r K8sProbeResolution) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO k8s_probe_resolutions (template_id, cluster_id, resolved_group, resolved_version, resolved_resource, is_override)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (template_id, cluster_id) DO UPDATE SET
		   resolved_group=$3, resolved_version=$4, resolved_resource=$5, is_override=$6, resolved_at=now()`,
		r.TemplateID, r.ClusterID, r.ResolvedGroup, r.ResolvedVersion, r.ResolvedResource, r.IsOverride)
	return err
}

func (s *Store) GetProbeResolution(ctx context.Context, templateID, clusterID uuid.UUID) (*K8sProbeResolution, error) {
	var r K8sProbeResolution
	err := s.pool.QueryRow(ctx,
		`SELECT id, template_id, cluster_id, resolved_group, resolved_version, resolved_resource, is_override, resolved_at
		 FROM k8s_probe_resolutions WHERE template_id=$1 AND cluster_id=$2`, templateID, clusterID).
		Scan(&r.ID, &r.TemplateID, &r.ClusterID, &r.ResolvedGroup, &r.ResolvedVersion, &r.ResolvedResource, &r.IsOverride, &r.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) ListProbeResolutions(ctx context.Context, clusterID uuid.UUID) ([]K8sProbeResolution, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, template_id, cluster_id, resolved_group, resolved_version, resolved_resource, is_override, resolved_at
		 FROM k8s_probe_resolutions WHERE cluster_id=$1`, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []K8sProbeResolution
	for rows.Next() {
		var r K8sProbeResolution
		if err := rows.Scan(&r.ID, &r.TemplateID, &r.ClusterID, &r.ResolvedGroup, &r.ResolvedVersion, &r.ResolvedResource, &r.IsOverride, &r.ResolvedAt); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (s *Store) RotateClusterCredential(ctx context.Context, clusterID uuid.UUID, credType, secretRef string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_cluster_credentials SET is_active=false, revoked_at=now() WHERE cluster_id=$1 AND is_active=true`, clusterID)
	if err != nil {
		return fmt.Errorf("revoke old credentials: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO k8s_cluster_credentials (cluster_id, credential_type, secret_ref, is_active) VALUES ($1, $2, $3, true)`,
		clusterID, credType, secretRef)
	return err
}

func (s *Store) GetActiveClusterCredential(ctx context.Context, clusterID uuid.UUID) (string, string, error) {
	var credType, secretRef string
	err := s.pool.QueryRow(ctx,
		`SELECT credential_type, secret_ref FROM k8s_cluster_credentials WHERE cluster_id=$1 AND is_active=true ORDER BY issued_at DESC LIMIT 1`,
		clusterID).Scan(&credType, &secretRef)
	if err != nil {
		return "", "", err
	}
	return credType, secretRef, nil
}

func (s *Store) GetStaleHandshakeClusters(ctx context.Context, maxAge time.Duration) ([]uuid.UUID, error) {
	cutoff := time.Now().Add(-maxAge)
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT cluster_id FROM k8s_cluster_handshakes
		 WHERE status IN ('registered','connected') AND last_heartbeat < $1`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
