package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type IncidentManager struct {
	store *store.Store
}

func NewIncidentManager(s *store.Store) *IncidentManager {
	return &IncidentManager{store: s}
}

func operationalDegradation(adapter string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"kind": "operational", "adapter": adapter})
	return b
}

func qosDegradation(adapter string, passRate float64, qosLevel string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"kind":      "qos",
		"adapter":   adapter,
		"pass_rate": passRate,
		"qos_level": qosLevel,
	})
	return b
}

func (m *IncidentManager) OnDown(ctx context.Context, targetType string, targetID uuid.UUID, telemetryID *uuid.UUID, adapter string) {
	deg := operationalDegradation(adapter)

	existing, err := m.store.GetOpenIncident(ctx, targetType, targetID)
	if err != nil && err != pgx.ErrNoRows {
		slog.Error("incident: check open", "error", err)
		return
	}

	if existing != nil {
		_ = m.store.UpdateIncidentDegradationAndSource(ctx, existing.ID, telemetryID, deg)
		if existing.Status != "investigating" && existing.Status != "identified" {
			_ = m.store.UpdateIncidentStatus(ctx, existing.ID, "investigating", nil)
			_ = m.store.CreateIncidentUpdate(ctx, existing.ID, "investigating",
				"Service is experiencing issues again.", "system")
		}
		return
	}

	title := fmt.Sprintf("Disruption with %s", targetType)
	inc, err := m.store.CreateIncident(ctx, targetType, targetID, title, "major", telemetryID, deg)
	if err != nil {
		slog.Error("incident: create", "error", err)
		return
	}

	_ = m.store.CreateIncidentUpdate(ctx, inc.ID, "investigating",
		"We are investigating reports of impacted performance.", "system")

	slog.Info("incident opened", "incident_id", inc.ID, "target_type", targetType, "target_id", targetID)
}

func (m *IncidentManager) OnDegraded(ctx context.Context, targetType string, targetID uuid.UUID, telemetryID *uuid.UUID, adapter string, passRate float64, qosLevel string) {
	deg := qosDegradation(adapter, passRate, qosLevel)

	existing, err := m.store.GetOpenIncident(ctx, targetType, targetID)
	if err != nil && err != pgx.ErrNoRows {
		slog.Error("incident: check open", "error", err)
		return
	}

	if existing != nil {
		_ = m.store.UpdateIncidentDegradationAndSource(ctx, existing.ID, telemetryID, deg)
		_ = m.store.CreateIncidentUpdate(ctx, existing.ID, "identified",
			fmt.Sprintf("Quality of service degraded. Pass rate: %.1f%%", passRate), "system")
		_ = m.store.UpdateIncidentStatus(ctx, existing.ID, "identified", nil)
		return
	}

	title := fmt.Sprintf("Degraded performance for %s", targetType)
	inc, err := m.store.CreateIncident(ctx, targetType, targetID, title, "minor", telemetryID, deg)
	if err != nil {
		slog.Error("incident: create degraded", "error", err)
		return
	}

	_ = m.store.CreateIncidentUpdate(ctx, inc.ID, "investigating",
		fmt.Sprintf("Quality of service degraded. Pass rate: %.1f%%", passRate), "system")

	slog.Info("degraded incident opened", "incident_id", inc.ID, "target_type", targetType, "target_id", targetID)
}

func (m *IncidentManager) OnRecovery(ctx context.Context, targetType string, targetID uuid.UUID) {
	existing, err := m.store.GetOpenIncident(ctx, targetType, targetID)
	if err != nil {
		return
	}

	if existing == nil || existing.Status == "resolved" {
		return
	}

	if existing.Status == "monitoring" {
		rb := "system"
		_ = m.store.UpdateIncidentStatus(ctx, existing.ID, "resolved", &rb)
		_ = m.store.CreateIncidentUpdate(ctx, existing.ID, "resolved",
			"This incident has been resolved.", "system")
		slog.Info("incident resolved", "incident_id", existing.ID)
		return
	}

	_ = m.store.UpdateIncidentStatus(ctx, existing.ID, "monitoring", nil)
	_ = m.store.CreateIncidentUpdate(ctx, existing.ID, "monitoring",
		"A fix has been implemented and we are monitoring the results.", "system")
}
