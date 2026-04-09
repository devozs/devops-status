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

func (e *StatusEvaluator) Evaluate(ctx context.Context, targetType string, targetID uuid.UUID, probeKind string, windowSize, failuresDown, successUp int) {
	samples, err := e.store.GetRecentSamples(ctx, targetType, targetID, probeKind, windowSize)
	if err != nil {
		slog.Error("evaluate: fetch samples", "error", err)
		return
	}

	if len(samples) == 0 {
		return
	}

	switch probeKind {
	case "operational":
		e.evaluateOperational(ctx, targetType, targetID, samples, failuresDown, successUp)
	case "qos":
		e.evaluateQoS(ctx, targetType, targetID, samples)
	}
}

func (e *StatusEvaluator) evaluateOperational(ctx context.Context, targetType string, targetID uuid.UUID, samples []store.SampleRecord, failuresDown, successUp int) {
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
		e.incident.OnDown(ctx, targetType, targetID)
	} else if consecutiveSuccess >= successUp {
		e.incident.OnRecovery(ctx, targetType, targetID)
	}
}

func (e *StatusEvaluator) evaluateQoS(ctx context.Context, targetType string, targetID uuid.UUID, samples []store.SampleRecord) {
	if len(samples) == 0 {
		return
	}

	successCount := 0
	for _, s := range samples {
		if s.Success {
			successCount++
		}
	}

	passRate := float64(successCount) / float64(len(samples)) * 100

	var level string
	if passRate >= 99 {
		level = "green"
	} else if passRate >= 95 {
		level = "yellow"
	} else {
		level = "red"
	}

	slog.Debug("qos evaluation", "target_type", targetType, "target_id", targetID, "pass_rate", passRate, "level", level)

	if level == "red" {
		e.incident.OnDegraded(ctx, targetType, targetID, passRate)
	} else if level == "green" {
		e.incident.OnRecovery(ctx, targetType, targetID)
	}
}
