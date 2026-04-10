package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListEnvironments(ctx context.Context, publicOnly bool) ([]model.Environment, error) {
	query := `SELECT id, name, slug, description, env_type, is_public, criticality, k8s_cluster_id, created_at, updated_at FROM environments`
	if publicOnly {
		query += ` WHERE is_public = true`
	}
	query += ` ORDER BY name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	envs := make([]model.Environment, 0)
	for rows.Next() {
		var env model.Environment
		if err := rows.Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.K8sClusterID, &env.CreatedAt, &env.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		envs = append(envs, env)
	}
	return envs, nil
}

func (s *Store) GetEnvironmentByID(ctx context.Context, id uuid.UUID) (*model.Environment, error) {
	var env model.Environment
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, env_type, is_public, criticality, k8s_cluster_id, created_at, updated_at
		 FROM environments WHERE id = $1`, id).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.K8sClusterID, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}
	return &env, nil
}

func (s *Store) GetEnvironmentBySlug(ctx context.Context, slug string) (*model.Environment, error) {
	var env model.Environment
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, env_type, is_public, criticality, k8s_cluster_id, created_at, updated_at
		 FROM environments WHERE slug = $1`, slug).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.K8sClusterID, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get environment by slug: %w", err)
	}
	return &env, nil
}

// EnvironmentSlugExists reports whether another row already uses this slug (excluding excludeID when set).
func (s *Store) EnvironmentSlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	var found uuid.UUID
	var err error
	if excludeID != nil {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM environments WHERE slug = $1 AND id <> $2 LIMIT 1`,
			slug, *excludeID).Scan(&found)
	} else {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM environments WHERE slug = $1 LIMIT 1`,
			slug).Scan(&found)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("environment slug lookup: %w", err)
	}
	return true, nil
}

type CreateEnvironmentInput struct {
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	Description    string     `json:"description"`
	EnvType        string     `json:"env_type"`
	IsPublic       bool       `json:"is_public"`
	Criticality    string     `json:"criticality"`
	K8sClusterID   *uuid.UUID `json:"k8s_cluster_id,omitempty"`
}

func (s *Store) CreateEnvironment(ctx context.Context, input CreateEnvironmentInput) (*model.Environment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var env model.Environment
	err = tx.QueryRow(ctx,
		`INSERT INTO environments (name, slug, description, env_type, is_public, criticality, k8s_cluster_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, name, slug, description, env_type, is_public, criticality, k8s_cluster_id, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.EnvType, input.IsPublic, input.Criticality, input.K8sClusterID).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.K8sClusterID, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}

	if input.K8sClusterID != nil {
		_, err = tx.Exec(ctx,
			`UPDATE k8s_clusters SET environment_id = $1 WHERE id = $2`,
			env.ID, *input.K8sClusterID)
		if err != nil {
			return nil, fmt.Errorf("bind cluster to environment: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &env, nil
}

type UpdateEnvironmentInput struct {
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	Description    string     `json:"description"`
	EnvType        string     `json:"env_type"`
	IsPublic       bool       `json:"is_public"`
	Criticality    string     `json:"criticality"`
	K8sClusterID   *uuid.UUID `json:"k8s_cluster_id,omitempty"`
}

func (s *Store) UpdateEnvironment(ctx context.Context, id uuid.UUID, input UpdateEnvironmentInput) (*model.Environment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldPrimary *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT k8s_cluster_id FROM environments WHERE id = $1`, id).Scan(&oldPrimary)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("update environment: not found")
		}
		return nil, fmt.Errorf("load environment: %w", err)
	}

	// Detach previous primary from this environment when changing or clearing.
	if oldPrimary != nil {
		changing := input.K8sClusterID == nil || *input.K8sClusterID != *oldPrimary
		if changing {
			_, err = tx.Exec(ctx,
				`UPDATE k8s_clusters SET environment_id = NULL WHERE id = $1 AND environment_id = $2`,
				*oldPrimary, id)
			if err != nil {
				return nil, fmt.Errorf("clear old cluster binding: %w", err)
			}
		}
	}

	var env model.Environment
	now := time.Now()
	err = tx.QueryRow(ctx,
		`UPDATE environments SET name=$1, slug=$2, description=$3, env_type=$4, is_public=$5, criticality=$6, k8s_cluster_id=$7, updated_at=$8
		 WHERE id=$9
		 RETURNING id, name, slug, description, env_type, is_public, criticality, k8s_cluster_id, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.EnvType, input.IsPublic, input.Criticality, input.K8sClusterID, now, id).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.K8sClusterID, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update environment: %w", err)
	}

	if input.K8sClusterID != nil {
		_, err = tx.Exec(ctx,
			`UPDATE k8s_clusters SET environment_id = $1 WHERE id = $2`,
			id, *input.K8sClusterID)
		if err != nil {
			return nil, fmt.Errorf("bind cluster to environment: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &env, nil
}

func (s *Store) DeleteEnvironment(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM environments WHERE id = $1`, id)
	return err
}
