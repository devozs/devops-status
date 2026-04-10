package engine

import (
	"context"
	"log/slog"

	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

type StatusEvaluator struct {
	store    *store.Store
	incident *IncidentManager
}

func NewStatusEvaluator(s *store.Store, inc *IncidentManager) *StatusEvaluator {
	return &StatusEvaluator{store: s, incident: inc}
}

func (e *StatusEvaluator) Evaluate(ctx context.Context, targetType string, targetID uuid.UUID, probeKind string, windowSize, failuresDown, successUp int, adapter string, qosThresholds []byte, telemetryID *uuid.UUID) {
	samples, err := e.store.GetRecentSamplesForProbe(ctx, targetType, targetID, telemetryID, probeKind, windowSize)
	if err != nil {
		slog.Error("evaluate: fetch samples", "error", err)
		return
	}

	if len(samples) == 0 {
		return
	}

	switch probeKind {
	case "operational":
		e.evaluateOperational(ctx, targetType, targetID, telemetryID, adapter, samples, failuresDown, successUp)
	case "qos":
		e.evaluateQoS(ctx, targetType, targetID, telemetryID, adapter, qosThresholds, samples)
	}
}

func (e *StatusEvaluator) evaluateOperational(ctx context.Context, targetType string, targetID uuid.UUID, telemetryID *uuid.UUID, adapter string, samples []store.SampleRecord, failuresDown, successUp int) {
	consecutiveFails := 0
	consecutiveSuccess := 0

	for _, s := range samples {
		if !s.Success {
			consecutiveFails++
			consecutiveSuccess = 0
		} else {
			consecutiveSuccess++
			consecutiveFails = 0
		}
	}

	if consecutiveFails >= failuresDown {
		e.incident.OnDown(ctx, targetType, targetID, telemetryID, adapter)
	} else if consecutiveSuccess >= successUp {
		e.incident.OnRecovery(ctx, targetType, targetID)
	}
}

func (e *StatusEvaluator) evaluateQoS(ctx context.Context, targetType string, targetID uuid.UUID, telemetryID *uuid.UUID, adapter string, qosThresholds []byte, samples []store.SampleRecord) {
	if len(samples) == 0 {
		return
	}

	level, passRate := e.evaluateQoSWithThresholds(samples, adapter, qosThresholds)

	slog.Debug("qos evaluation", "target_type", targetType, "target_id", targetID, "pass_rate", passRate, "level", level, "adapter", adapter)

	if level == "red" {
		e.incident.OnDegraded(ctx, targetType, targetID, telemetryID, adapter, passRate, level)
	} else if level == "green" {
		e.incident.OnRecovery(ctx, targetType, targetID)
	}
}
