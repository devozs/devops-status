package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) CreateAuditLog(ctx context.Context, userID *uuid.UUID, action, entityType string, entityID *uuid.UUID, details any, ipAddress string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO audit_logs (user_id, action, entity_type, entity_id, details_json, ip_address)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, action, entityType, entityID, detailsJSON, ipAddress)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}
