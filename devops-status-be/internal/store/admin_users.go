package store

import (
	"context"
	"fmt"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func (s *Store) GetAdminUserByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var u model.AdminUser
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, is_active, last_login, created_at, updated_at
		 FROM admin_users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get admin user by username: %w", err)
	}
	return &u, nil
}

func (s *Store) GetAdminUserByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error) {
	var u model.AdminUser
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, is_active, last_login, created_at, updated_at
		 FROM admin_users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get admin user by id: %w", err)
	}
	return &u, nil
}

func (s *Store) UpdateAdminUserLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE admin_users SET last_login = $1, updated_at = $1 WHERE id = $2`, now, id)
	return err
}
