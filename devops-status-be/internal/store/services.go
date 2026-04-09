package store

import (
	"context"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) ListServices(ctx context.Context, publicOnly bool) ([]model.Service, error) {
	query := `SELECT id, name, slug, description, is_public, criticality, created_at, updated_at FROM services`
	if publicOnly {
		query += ` WHERE is_public = true`
	}
	query += ` ORDER BY name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var services []model.Service
	for rows.Next() {
		var svc model.Service
		if err := rows.Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		services = append(services, svc)
	}
	return services, nil
}

func (s *Store) GetServiceByID(ctx context.Context, id uuid.UUID) (*model.Service, error) {
	var svc model.Service
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, is_public, criticality, created_at, updated_at
		 FROM services WHERE id = $1`, id).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	return &svc, nil
}

func (s *Store) GetServiceBySlug(ctx context.Context, slug string) (*model.Service, error) {
	var svc model.Service
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, description, is_public, criticality, created_at, updated_at
		 FROM services WHERE slug = $1`, slug).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get service by slug: %w", err)
	}
	return &svc, nil
}

type CreateServiceInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	Criticality string `json:"criticality"`
}

func (s *Store) CreateService(ctx context.Context, input CreateServiceInput) (*model.Service, error) {
	var svc model.Service
	err := s.pool.QueryRow(ctx,
		`INSERT INTO services (name, slug, description, is_public, criticality)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, slug, description, is_public, criticality, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.IsPublic, input.Criticality).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}
	return &svc, nil
}

type UpdateServiceInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	Criticality string `json:"criticality"`
}

func (s *Store) UpdateService(ctx context.Context, id uuid.UUID, input UpdateServiceInput) (*model.Service, error) {
	var svc model.Service
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE services SET name=$1, slug=$2, description=$3, is_public=$4, criticality=$5, updated_at=$6
		 WHERE id=$7
		 RETURNING id, name, slug, description, is_public, criticality, created_at, updated_at`,
		input.Name, input.Slug, input.Description, input.IsPublic, input.Criticality, now, id).
		Scan(&svc.ID, &svc.Name, &svc.Slug, &svc.Description, &svc.IsPublic, &svc.Criticality, &svc.CreatedAt, &svc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update service: %w", err)
	}
	return &svc, nil
}

func (s *Store) DeleteService(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	return err
}
