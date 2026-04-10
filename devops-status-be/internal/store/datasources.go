package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) ListDataSources(ctx context.Context) ([]model.DataSource, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, ds_type, adapter, config_json, secret_ref, timeout_ms, retries, created_at, updated_at
		 FROM data_sources ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list data sources: %w", err)
	}
	defer rows.Close()

	sources := make([]model.DataSource, 0)
	for rows.Next() {
		var ds model.DataSource
		var cfgBytes []byte
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.DSType, &ds.Adapter, &cfgBytes, &ds.SecretRef, &ds.TimeoutMs, &ds.Retries, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan data source: %w", err)
		}
		json.Unmarshal(cfgBytes, &ds.ConfigJSON)
		sources = append(sources, ds)
	}
	return sources, nil
}

func (s *Store) GetDataSourceByID(ctx context.Context, id uuid.UUID) (*model.DataSource, error) {
	var ds model.DataSource
	var cfgBytes []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, ds_type, adapter, config_json, secret_ref, timeout_ms, retries, created_at, updated_at
		 FROM data_sources WHERE id = $1`, id).
		Scan(&ds.ID, &ds.Name, &ds.DSType, &ds.Adapter, &cfgBytes, &ds.SecretRef, &ds.TimeoutMs, &ds.Retries, &ds.CreatedAt, &ds.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get data source: %w", err)
	}
	json.Unmarshal(cfgBytes, &ds.ConfigJSON)
	return &ds, nil
}

type CreateDataSourceInput struct {
	Name       string `json:"name"`
	DSType     string `json:"ds_type"`
	Adapter    string `json:"adapter"`
	ConfigJSON any    `json:"config_json"`
	SecretRef  string `json:"secret_ref"`
	TimeoutMs  int    `json:"timeout_ms"`
	Retries    int    `json:"retries"`
}

func (s *Store) CreateDataSource(ctx context.Context, input CreateDataSourceInput) (*model.DataSource, error) {
	cfgBytes, _ := json.Marshal(input.ConfigJSON)
	var ds model.DataSource
	var cfgOut []byte
	err := s.pool.QueryRow(ctx,
		`INSERT INTO data_sources (name, ds_type, adapter, config_json, secret_ref, timeout_ms, retries)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, name, ds_type, adapter, config_json, secret_ref, timeout_ms, retries, created_at, updated_at`,
		input.Name, input.DSType, input.Adapter, cfgBytes, input.SecretRef, input.TimeoutMs, input.Retries).
		Scan(&ds.ID, &ds.Name, &ds.DSType, &ds.Adapter, &cfgOut, &ds.SecretRef, &ds.TimeoutMs, &ds.Retries, &ds.CreatedAt, &ds.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create data source: %w", err)
	}
	json.Unmarshal(cfgOut, &ds.ConfigJSON)
	return &ds, nil
}

func (s *Store) UpdateDataSource(ctx context.Context, id uuid.UUID, input CreateDataSourceInput) (*model.DataSource, error) {
	cfgBytes, _ := json.Marshal(input.ConfigJSON)
	var ds model.DataSource
	var cfgOut []byte
	now := time.Now()
	err := s.pool.QueryRow(ctx,
		`UPDATE data_sources SET name=$1, ds_type=$2, adapter=$3, config_json=$4, secret_ref=$5, timeout_ms=$6, retries=$7, updated_at=$8
		 WHERE id=$9
		 RETURNING id, name, ds_type, adapter, config_json, secret_ref, timeout_ms, retries, created_at, updated_at`,
		input.Name, input.DSType, input.Adapter, cfgBytes, input.SecretRef, input.TimeoutMs, input.Retries, now, id).
		Scan(&ds.ID, &ds.Name, &ds.DSType, &ds.Adapter, &cfgOut, &ds.SecretRef, &ds.TimeoutMs, &ds.Retries, &ds.CreatedAt, &ds.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update data source: %w", err)
	}
	json.Unmarshal(cfgOut, &ds.ConfigJSON)
	return &ds, nil
}

func (s *Store) DeleteDataSource(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM data_sources WHERE id = $1`, id)
	return err
}
