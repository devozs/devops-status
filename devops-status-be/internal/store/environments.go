package store

import (
	"context"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) ListEnvironments(ctx context.Context, publicOnly bool) ([]model.Environment, error) {
	query := `SELECT id, name, slug, description, env_type, is_public, criticality, created_at, updated_at FROM environments`
	if publicOnly {
		query += ` WHERE is_public = true`
	}
	query += ` ORDER BY name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	var envs []model.Environment
	for rows.Next() {
		var env model.Environment
		if err := rows.Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.CreatedAt, &env.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		envs = append(envs, env)
	}
	return envs, nil
}

func (s *Store) GetEnvironmentByID(ctx context.Context, id uuid.UUID) (*model.Environment, error) {
	var env model.Environment
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, env_type, is_public, criticality, created_at, updated_at
		 FROM environments WHERE id = $1`, id).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}
	return &env, nil
}

func (s *Store) GetEnvironmentBySlug(ctx context.Context, slug string) (*model.Environment, error) {
	var env model.Environment
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, env_type, is_public, criticality, created_at, updated_at
		 FROM environments WHERE slug = $1`, slug).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get environment by slug: %w", err)
	}
	return &env, nil
}

type CreateEnvironmentInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	EnvType     string `json:"env_type"`
	IsPublic    bool   `json:"is_public"`
	Criticality string `json:"criticality"`
}

func (s *Store) CreateEnvironment(ctx context.Context, input CreateEnvironmentInput) (*model.Environment, error) {
	var env model.Environment
	err := s.pool.QueryRow(ctx,
		`INSERT INTO environments (name, slug, description, env_type, is_public, criticality)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, slug, description, env_type, is_public, criticality, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.EnvType, input.IsPublic, input.Criticality).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return &env, nil
}

type UpdateEnvironmentInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	EnvType     string `json:"env_type"`
	IsPublic    bool   `json:"is_public"`
	Criticality string `json:"criticality"`
}

func (s *Store) UpdateEnvironment(ctx context.Context, id uuid.UUID, input UpdateEnvironmentInput) (*model.Environment, error) {
	var env model.Environment
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE environments SET name=$1, slug=$2, description=$3, env_type=$4, is_public=$5, criticality=$6, updated_at=$7
		 WHERE id=$8
		 RETURNING id, name, slug, description, env_type, is_public, criticality, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.EnvType, input.IsPublic, input.Criticality, now, id).
		Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.EnvType, &env.IsPublic, &env.Criticality, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update environment: %w", err)
	}
	return &env, nil
}

func (s *Store) DeleteEnvironment(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM environments WHERE id = $1`, id)
	return err
}

func (s *Store) ListEnvironmentMembers(ctx context.Context, envID uuid.UUID) ([]model.Service, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT s.id, s.name, s.slug, s.description, s.is_public, s.criticality, s.created_at, s.updated_at
		 FROM services s
		 JOIN environment_service_membership m ON m.service_id = s.id
		 WHERE m.environment_id = $1
		 ORDER BY s.name ASC`, envID)
	if err != nil {
		return nil, fmt.Errorf("list environment members: %w", err)
	}
	defer rows.Close()

	var services []model.Service
	for rows.Next() {
		var svc model.Service
		if err := rows.Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan member service: %w", err)
		}
		services = append(services, svc)
	}
	return services, nil
}

func (s *Store) AddEnvironmentMember(ctx context.Context, envID, svcID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO environment_service_membership (environment_id, service_id) VALUES ($1, $2)
		 ON CONFLICT (environment_id, service_id) DO NOTHING`, envID, svcID)
	return err
}

func (s *Store) RemoveEnvironmentMember(ctx context.Context, envID, svcID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM environment_service_membership WHERE environment_id = $1 AND service_id = $2`, envID, svcID)
	return err
}
