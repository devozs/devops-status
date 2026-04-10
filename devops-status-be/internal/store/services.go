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

func (s *Store) ListServices(ctx context.Context, publicOnly bool) ([]model.Service, error) {
	query := `SELECT id, name, slug, description, is_public, criticality, service_provider_id, created_at, updated_at FROM services`
	if publicOnly {
		query += ` WHERE is_public = true`
	}
	query += ` ORDER BY name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	services := make([]model.Service, 0)
	for rows.Next() {
		var svc model.Service
		if err := rows.Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.ServiceProviderID, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		services = append(services, svc)
	}
	return services, nil
}

func (s *Store) GetServiceByID(ctx context.Context, id uuid.UUID) (*model.Service, error) {
	var svc model.Service
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, is_public, criticality, service_provider_id, created_at, updated_at
		 FROM services WHERE id = $1`, id).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.ServiceProviderID, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	return &svc, nil
}

func (s *Store) GetServiceBySlug(ctx context.Context, slug string) (*model.Service, error) {
	var svc model.Service
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, is_public, criticality, service_provider_id, created_at, updated_at
		 FROM services WHERE slug = $1`, slug).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.ServiceProviderID, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get service by slug: %w", err)
	}
	return &svc, nil
}

// ServiceSlugExists reports whether another row already uses this slug (excluding excludeID when set).
func (s *Store) ServiceSlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	var found uuid.UUID
	var err error
	if excludeID != nil {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM services WHERE slug = $1 AND id <> $2 LIMIT 1`,
			slug, *excludeID).Scan(&found)
	} else {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM services WHERE slug = $1 LIMIT 1`,
			slug).Scan(&found)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("service slug lookup: %w", err)
	}
	return true, nil
}

type CreateServiceInput struct {
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	Description         string     `json:"description"`
	IsPublic            bool       `json:"is_public"`
	Criticality         string     `json:"criticality"`
	ServiceProviderID   *uuid.UUID `json:"service_provider_id,omitempty"`
}

func (s *Store) CreateService(ctx context.Context, input CreateServiceInput) (*model.Service, error) {
	var svc model.Service
	err := s.pool.QueryRow(ctx,
		`INSERT INTO services (name, slug, description, is_public, criticality, service_provider_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, slug, description, is_public, criticality, service_provider_id, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.IsPublic, input.Criticality, input.ServiceProviderID).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.ServiceProviderID, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}
	return &svc, nil
}

type UpdateServiceInput struct {
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	Description         string     `json:"description"`
	IsPublic            bool       `json:"is_public"`
	Criticality         string     `json:"criticality"`
	ServiceProviderID   *uuid.UUID `json:"service_provider_id,omitempty"`
}

func (s *Store) UpdateService(ctx context.Context, id uuid.UUID, input UpdateServiceInput) (*model.Service, error) {
	var svc model.Service
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE services SET name=$1, slug=$2, description=$3, is_public=$4, criticality=$5, service_provider_id=$6, updated_at=$7
		 WHERE id=$8
		 RETURNING id, name, slug, description, is_public, criticality, service_provider_id, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.IsPublic, input.Criticality, input.ServiceProviderID, now, id).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.ServiceProviderID, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update service: %w", err)
	}
	return &svc, nil
}

func (s *Store) DeleteService(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	return err
}
