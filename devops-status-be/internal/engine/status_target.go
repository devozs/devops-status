package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

func hasQoSBytes(q []byte) bool {
	return len(q) > 0 && string(q) != "{}" && string(q) != "null"
}

func operationalStreaks(samples []store.SampleRecord) (consecutiveFails, consecutiveSuccess int) {
	for _, s := range samples {
		if !s.Success {
			consecutiveFails++
			consecutiveSuccess = 0
		} else {
			consecutiveSuccess++
			consecutiveFails = 0
		}
	}
	return consecutiveFails, consecutiveSuccess
}

func telemetryLabel(t *model.Telemetry) string {
	if strings.TrimSpace(t.DisplayName) != "" {
		return t.DisplayName
	}
	return t.Name
}

// EvaluateTargetAfterProbe recomputes operational + QoS incident state for one target using all
// linked telemetry streams (avoids contradictory updates when workers finish probes concurrently).
func (e *StatusEvaluator) EvaluateTargetAfterProbe(ctx context.Context, targetType string, targetID uuid.UUID) {
	e.withTargetLock(targetType, targetID, func() {
		e.evaluateTargetIncidents(ctx, targetType, targetID)
	})
}

func (e *StatusEvaluator) evaluateTargetIncidents(ctx context.Context, targetType string, targetID uuid.UUID) {
	links, err := e.listTelemetryLinksForTarget(ctx, targetType, targetID)
	if err != nil {
		slog.Error("evaluate target: list links", "error", err, "target_type", targetType, "target_id", targetID)
		return
	}
	if len(links) == 0 {
		return
	}

	anyOpsDown := false
	everyOpsRecovered := true
	var downTID *uuid.UUID
	var downAdapter string

	for _, l := range links {
		t, err := e.store.GetTelemetryByID(ctx, l.telemetryID)
		if err != nil {
			everyOpsRecovered = false
			continue
		}
		samples, err := e.store.GetRecentSamplesForProbe(ctx, targetType, targetID, &l.telemetryID, "operational", l.windowSize)
		if err != nil {
			everyOpsRecovered = false
			continue
		}
		if len(samples) == 0 {
			everyOpsRecovered = false
			continue
		}
		cf, cs := operationalStreaks(samples)
		if cf >= l.failuresDown {
			anyOpsDown = true
			if downTID == nil {
				tid := l.telemetryID
				downTID = &tid
				downAdapter = t.Adapter
			}
		}
		if len(samples) < l.successUp || cs < l.successUp {
			everyOpsRecovered = false
		}
	}

	qosLinkCount := 0
	anyQoSRed := false
	allQoSGreen := true
	var worst *qosWorstStream

	for _, l := range links {
		t, err := e.store.GetTelemetryByID(ctx, l.telemetryID)
		if err != nil {
			continue
		}
		qosBytes, _ := json.Marshal(t.QosThresholds)
		if !hasQoSBytes(qosBytes) {
			continue
		}
		qosLinkCount++
		samples, err := e.store.GetRecentSamplesForProbe(ctx, targetType, targetID, &l.telemetryID, "qos", l.windowSize)
		if err != nil || len(samples) == 0 {
			allQoSGreen = false
			continue
		}
		level, passRate := e.evaluateQoSWithThresholds(samples, t.Adapter, qosBytes)
		slog.Debug("qos aggregate", "target_type", targetType, "target_id", targetID, "telemetry_id", l.telemetryID, "pass_rate", passRate, "level", level, "adapter", t.Adapter)
		if level == "red" {
			anyQoSRed = true
			allQoSGreen = false
			if worst == nil || passRate < worst.passRate {
				worst = &qosWorstStream{
					telemetryID: l.telemetryID,
					adapter:     t.Adapter,
					passRate:    passRate,
					label:       telemetryLabel(t),
				}
			}
		} else if level != "green" {
			allQoSGreen = false
		}
	}

	qosRecoveredOK := qosLinkCount == 0 || allQoSGreen

	if anyOpsDown {
		e.incident.OnDown(ctx, targetType, targetID, downTID, downAdapter)
		return
	}
	if anyQoSRed && worst != nil {
		hint := ""
		if qosLinkCount > 1 && worst.label != "" {
			hint = worst.label
		}
		e.incident.OnDegraded(ctx, targetType, targetID, &worst.telemetryID, worst.adapter, worst.passRate, "red", hint)
		return
	}
	if everyOpsRecovered && qosRecoveredOK {
		e.incident.OnRecovery(ctx, targetType, targetID)
	}
}

type qosWorstStream struct {
	telemetryID uuid.UUID
	adapter     string
	passRate    float64
	label       string
}

type evalLink struct {
	telemetryID uuid.UUID
	windowSize  int
	failuresDown int
	successUp   int
}

func (e *StatusEvaluator) listTelemetryLinksForTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]evalLink, error) {
	switch targetType {
	case "service":
		rows, err := e.store.ListServiceTelemetryLinks(ctx, &targetID)
		if err != nil {
			return nil, err
		}
		out := make([]evalLink, 0, len(rows))
		for _, r := range rows {
			out = append(out, evalLink{
				telemetryID:  r.TelemetryID,
				windowSize:   r.WindowSize,
				failuresDown: r.ConsecutiveFailuresToDown,
				successUp:    r.ConsecutiveSuccessToRecover,
			})
		}
		return out, nil
	case "environment":
		rows, err := e.store.ListEnvironmentTelemetryLinks(ctx, &targetID)
		if err != nil {
			return nil, err
		}
		out := make([]evalLink, 0, len(rows))
		for _, r := range rows {
			out = append(out, evalLink{
				telemetryID:  r.TelemetryID,
				windowSize:   r.WindowSize,
				failuresDown: r.ConsecutiveFailuresToDown,
				successUp:    r.ConsecutiveSuccessToRecover,
			})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown target_type %q", targetType)
	}
}
