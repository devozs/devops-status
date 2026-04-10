package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

// LinkedTelemetryRow is telemetry linked to a service or environment (TargetID = service or env id).
type LinkedTelemetryRow struct {
	TargetID    uuid.UUID
	TelemetryID uuid.UUID
	Name        string
	DisplayName string
	Adapter     string
}

func (s *Store) ListServicesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Service, error) {
	out := make(map[uuid.UUID]model.Service)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, slug, description, is_public, criticality, service_provider_id, created_at, updated_at
		 FROM services WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("list services by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var svc model.Service
		if err := rows.Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.ServiceProviderID, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
			return nil, err
		}
		out[svc.ID] = svc
	}
	return out, rows.Err()
}

func (s *Store) ListEnvironmentsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Environment, error) {
	out := make(map[uuid.UUID]model.Environment)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, slug, description, env_type, is_public, criticality, k8s_cluster_id, created_at, updated_at
		 FROM environments WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("list environments by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var env model.Environment
		if err := rows.Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.K8sClusterID, &env.CreatedAt, &env.UpdatedAt); err != nil {
			return nil, err
		}
		out[env.ID] = env
	}
	return out, rows.Err()
}

func (s *Store) ListServiceProvidersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.ServiceProvider, error) {
	out := make(map[uuid.UUID]model.ServiceProvider)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, host, port, image_url, provider_type, config_json, created_at, updated_at
		 FROM service_providers WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("list service providers by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p model.ServiceProvider
		var cfg []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.ImageURL, &p.ProviderType, &cfg, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(cfg, &p.ConfigJSON)
		out[p.ID] = p
	}
	return out, rows.Err()
}

// ListLinkedTelemetryForTargets returns linked telemetry rows for the given service and environment ids.
func (s *Store) ListLinkedTelemetryForTargets(ctx context.Context, serviceIDs, environmentIDs []uuid.UUID) ([]LinkedTelemetryRow, error) {
	var out []LinkedTelemetryRow
	if len(serviceIDs) > 0 {
		r, err := s.pool.Query(ctx,
			`SELECT stl.service_id, t.id, t.name, t.display_name, t.adapter
			 FROM service_telemetry_links stl
			 INNER JOIN telemetry t ON t.id = stl.telemetry_id
			 WHERE stl.service_id = ANY($1::uuid[])`, serviceIDs)
		if err != nil {
			return nil, fmt.Errorf("linked telemetry services: %w", err)
		}
		for r.Next() {
			var row LinkedTelemetryRow
			var dn sql.NullString
			if err := r.Scan(&row.TargetID, &row.TelemetryID, &row.Name, &dn, &row.Adapter); err != nil {
				r.Close()
				return nil, err
			}
			if dn.Valid && dn.String != "" {
				row.DisplayName = dn.String
			} else {
				row.DisplayName = row.Name
			}
			out = append(out, row)
		}
		r.Close()
	}
	if len(environmentIDs) > 0 {
		r, err := s.pool.Query(ctx,
			`SELECT etl.environment_id, t.id, t.name, t.display_name, t.adapter
			 FROM environment_telemetry_links etl
			 INNER JOIN telemetry t ON t.id = etl.telemetry_id
			 WHERE etl.environment_id = ANY($1::uuid[])`, environmentIDs)
		if err != nil {
			return nil, fmt.Errorf("linked telemetry environments: %w", err)
		}
		for r.Next() {
			var row LinkedTelemetryRow
			var dn sql.NullString
			if err := r.Scan(&row.TargetID, &row.TelemetryID, &row.Name, &dn, &row.Adapter); err != nil {
				r.Close()
				return nil, err
			}
			if dn.Valid && dn.String != "" {
				row.DisplayName = dn.String
			} else {
				row.DisplayName = row.Name
			}
			out = append(out, row)
		}
		r.Close()
	}
	return out, nil
}

func (s *Store) ListK8sClustersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]K8sCluster, error) {
	out := make(map[uuid.UUID]K8sCluster)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, environment_id, endpoint, auth_method, default_namespace, k8s_version, status, last_capability_scan,
			api_verify_ok, api_verify_checked_at, api_verify_detail, created_at, updated_at
		 FROM k8s_clusters WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("list k8s clusters by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c K8sCluster
		var apiOk sql.NullBool
		var apiAt sql.NullTime
		var apiDetail sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.EnvironmentID, &c.Endpoint, &c.AuthMethod, &c.DefaultNamespace,
			&c.K8sVersion, &c.Status, &c.LastCapabilityScan,
			&apiOk, &apiAt, &apiDetail,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if apiOk.Valid {
			v := apiOk.Bool
			c.APIVerifyOk = &v
		}
		if apiAt.Valid {
			t := apiAt.Time
			c.APIVerifyCheckedAt = &t
		}
		if apiDetail.Valid {
			c.APIVerifyDetail = apiDetail.String
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}
