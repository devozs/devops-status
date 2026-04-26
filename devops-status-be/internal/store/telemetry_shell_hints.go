package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListTelemetryShellHints(ctx context.Context, kind *string) ([]model.TelemetryShellHint, error) {
	var rows pgx.Rows
	var err error
	if kind != nil && strings.TrimSpace(*kind) != "" {
		rows, err = s.pool.Query(ctx,
			`SELECT id, kind, title, body, description, sort_order, created_at, updated_at
			 FROM telemetry_shell_hints WHERE kind = $1 ORDER BY sort_order ASC, title ASC`,
			strings.TrimSpace(*kind))
	} else {
		rows, err = s.pool.Query(ctx,
			`SELECT id, kind, title, body, description, sort_order, created_at, updated_at
			 FROM telemetry_shell_hints ORDER BY kind ASC, sort_order ASC, title ASC`)
	}
	if err != nil {
		return nil, fmt.Errorf("list telemetry shell hints: %w", err)
	}
	defer rows.Close()

	out := make([]model.TelemetryShellHint, 0)
	for rows.Next() {
		var h model.TelemetryShellHint
		if err := rows.Scan(&h.ID, &h.Kind, &h.Title, &h.Body, &h.Description, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan shell hint: %w", err)
		}
		out = append(out, h)
	}
	return out, nil
}

func (s *Store) GetTelemetryShellHintByID(ctx context.Context, id uuid.UUID) (*model.TelemetryShellHint, error) {
	var h model.TelemetryShellHint
	err := s.pool.QueryRow(ctx,
		`SELECT id, kind, title, body, description, sort_order, created_at, updated_at
		 FROM telemetry_shell_hints WHERE id = $1`, id).
		Scan(&h.ID, &h.Kind, &h.Title, &h.Body, &h.Description, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get shell hint: %w", err)
	}
	return &h, nil
}

type CreateTelemetryShellHintInput struct {
	Kind        string
	Title       string
	Body        string
	Description string
	SortOrder   *int
}

func (s *Store) nextShellHintSortOrder(ctx context.Context, tx pgx.Tx, kind string) (int, error) {
	var next int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM telemetry_shell_hints WHERE kind = $1`, kind).Scan(&next)
	return next, err
}

func (s *Store) CreateTelemetryShellHint(ctx context.Context, in CreateTelemetryShellHintInput) (*model.TelemetryShellHint, error) {
	title := strings.TrimSpace(in.Title)
	kind := strings.TrimSpace(in.Kind)
	body := in.Body
	desc := strings.TrimSpace(in.Description)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	} else {
		sortOrder, err = s.nextShellHintSortOrder(ctx, tx, kind)
		if err != nil {
			return nil, err
		}
	}

	var h model.TelemetryShellHint
	err = tx.QueryRow(ctx,
		`INSERT INTO telemetry_shell_hints (kind, title, body, description, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, kind, title, body, description, sort_order, created_at, updated_at`,
		kind, title, body, desc, sortOrder).
		Scan(&h.ID, &h.Kind, &h.Title, &h.Body, &h.Description, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert shell hint: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &h, nil
}

type UpdateTelemetryShellHintInput struct {
	Kind        string
	Title       string
	Body        string
	Description string
	SortOrder   *int
}

func (s *Store) UpdateTelemetryShellHint(ctx context.Context, id uuid.UUID, in UpdateTelemetryShellHintInput) (*model.TelemetryShellHint, error) {
	title := strings.TrimSpace(in.Title)
	kind := strings.TrimSpace(in.Kind)
	body := in.Body
	desc := strings.TrimSpace(in.Description)

	var sortOrder any
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	} else {
		sortOrder = nil
	}

	var h model.TelemetryShellHint
	var err error
	if sortOrder != nil {
		err = s.pool.QueryRow(ctx,
			`UPDATE telemetry_shell_hints SET kind = $2, title = $3, body = $4, description = $5, sort_order = $6, updated_at = now()
			 WHERE id = $1
			 RETURNING id, kind, title, body, description, sort_order, created_at, updated_at`,
			id, kind, title, body, desc, sortOrder).
			Scan(&h.ID, &h.Kind, &h.Title, &h.Body, &h.Description, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt)
	} else {
		err = s.pool.QueryRow(ctx,
			`UPDATE telemetry_shell_hints SET kind = $2, title = $3, body = $4, description = $5, updated_at = now()
			 WHERE id = $1
			 RETURNING id, kind, title, body, description, sort_order, created_at, updated_at`,
			id, kind, title, body, desc).
			Scan(&h.ID, &h.Kind, &h.Title, &h.Body, &h.Description, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update shell hint: %w", err)
	}
	return &h, nil
}

func (s *Store) DeleteTelemetryShellHint(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM telemetry_shell_hints WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete shell hint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListTelemetryShellHintIDsForKind returns all hint ids for kind in sort_order (for reorder validation).
func (s *Store) ListTelemetryShellHintIDsForKind(ctx context.Context, kind string) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id FROM telemetry_shell_hints WHERE kind = $1 ORDER BY sort_order ASC, title ASC`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// ReorderTelemetryShellHints sets sort_order to index for orderedIDs (full permutation for kind).
func (s *Store) ReorderTelemetryShellHints(ctx context.Context, kind string, orderedIDs []uuid.UUID) error {
	if len(orderedIDs) == 0 {
		return fmt.Errorf("ordered_ids is empty")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := s.listShellHintIDsForKindTx(ctx, tx, kind)
	if err != nil {
		return err
	}
	if len(current) != len(orderedIDs) {
		return fmt.Errorf("ordered_ids length %d does not match hint count %d for kind", len(orderedIDs), len(current))
	}
	want := make(map[uuid.UUID]bool, len(current))
	for _, id := range current {
		want[id] = true
	}
	seen := make(map[uuid.UUID]bool, len(orderedIDs))
	for _, id := range orderedIDs {
		if !want[id] {
			return fmt.Errorf("unknown id %s for kind %s", id, kind)
		}
		if seen[id] {
			return fmt.Errorf("duplicate id in ordered_ids")
		}
		seen[id] = true
	}
	for i, id := range orderedIDs {
		if _, err := tx.Exec(ctx,
			`UPDATE telemetry_shell_hints SET sort_order = $2, updated_at = now() WHERE id = $1 AND kind = $3`,
			id, i, kind); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) listShellHintIDsForKindTx(ctx context.Context, tx pgx.Tx, kind string) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx,
		`SELECT id FROM telemetry_shell_hints WHERE kind = $1 ORDER BY sort_order ASC, title ASC`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
