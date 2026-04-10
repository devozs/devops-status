package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListServiceProviders(ctx context.Context) ([]model.ServiceProvider, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, host, port, image_url, provider_type, config_json, created_at, updated_at
		 FROM service_providers ORDER BY name ASC, host ASC`)
	if err != nil {
		return nil, fmt.Errorf("list service providers: %w", err)
	}
	defer rows.Close()

	out := make([]model.ServiceProvider, 0)
	for rows.Next() {
		var p model.ServiceProvider
		var cfgBytes []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.ImageURL, &p.ProviderType, &cfgBytes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan service provider: %w", err)
		}
		_ = json.Unmarshal(cfgBytes, &p.ConfigJSON)
		out = append(out, p)
	}
	return out, nil
}

func (s *Store) GetServiceProviderByID(ctx context.Context, id uuid.UUID) (*model.ServiceProvider, error) {
	var p model.ServiceProvider
	var cfgBytes []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, host, port, image_url, provider_type, config_json, created_at, updated_at
		 FROM service_providers WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.ImageURL, &p.ProviderType, &cfgBytes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get service provider: %w", err)
	}
	_ = json.Unmarshal(cfgBytes, &p.ConfigJSON)
	return &p, nil
}

// ServiceProviderNameExists reports whether another row uses the same name (case-insensitive), excluding excludeID.
func (s *Store) ServiceProviderNameExists(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	var found uuid.UUID
	var err error
	if excludeID != nil {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM service_providers WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) AND id <> $2 LIMIT 1`,
			name, *excludeID).Scan(&found)
	} else {
		err = s.pool.QueryRow(ctx,
			`SELECT id FROM service_providers WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) LIMIT 1`,
			name).Scan(&found)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("service provider name lookup: %w", err)
	}
	return true, nil
}

type CreateServiceProviderInput struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ImageURL     string `json:"image_url"`
	ProviderType string `json:"provider_type"`
	ConfigJSON   any    `json:"config_json"`
}

func (s *Store) CreateServiceProvider(ctx context.Context, input CreateServiceProviderInput) (*model.ServiceProvider, error) {
	pt := normalizeProviderType(input.ProviderType)
	cfgBytes, err := json.Marshal(input.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("config json: %w", err)
	}
	if len(cfgBytes) == 0 || string(cfgBytes) == "null" {
		cfgBytes = []byte("{}")
	}
	var p model.ServiceProvider
	var cfgOut []byte
	err = s.pool.QueryRow(ctx,
		`INSERT INTO service_providers (name, host, port, image_url, provider_type, config_json)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, host, port, image_url, provider_type, config_json, created_at, updated_at`,
		input.Name, input.Host, input.Port, input.ImageURL, pt, cfgBytes).
		Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.ImageURL, &p.ProviderType, &cfgOut, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create service provider: %w", err)
	}
	_ = json.Unmarshal(cfgOut, &p.ConfigJSON)
	return &p, nil
}

type UpdateServiceProviderInput struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ImageURL     string `json:"image_url"`
	ProviderType string `json:"provider_type"`
	ConfigJSON   any    `json:"config_json"`
}

func (s *Store) UpdateServiceProvider(ctx context.Context, id uuid.UUID, input UpdateServiceProviderInput) (*model.ServiceProvider, error) {
	pt := normalizeProviderType(input.ProviderType)
	cfgBytes, err := json.Marshal(input.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("config json: %w", err)
	}
	if len(cfgBytes) == 0 || string(cfgBytes) == "null" {
		cfgBytes = []byte("{}")
	}
	var p model.ServiceProvider
	now := time.Now()
	var cfgOut []byte
	err = s.pool.QueryRow(ctx,
		`UPDATE service_providers SET name=$1, host=$2, port=$3, image_url=$4, provider_type=$5, config_json=$6, updated_at=$7 WHERE id=$8
		 RETURNING id, name, host, port, image_url, provider_type, config_json, created_at, updated_at`,
		input.Name, input.Host, input.Port, input.ImageURL, pt, cfgBytes, now, id).
		Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.ImageURL, &p.ProviderType, &cfgOut, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update service provider: %w", err)
	}
	_ = json.Unmarshal(cfgOut, &p.ConfigJSON)
	return &p, nil
}

func normalizeProviderType(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "tcp"
	}
	return s
}

func (s *Store) DeleteServiceProvider(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM service_providers WHERE id = $1`, id)
	return err
}
