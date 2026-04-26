package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/devops-status/be/internal/shellhints"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

var shellHintSlotKinds = map[string]string{
	"job_env_hint_id":           shellhints.KindJobEnv,
	"job_node_selector_hint_id": shellhints.KindJobNodeSelector,
	"container_prep_hint_id":    shellhints.KindContainerPrep,
	"probe_shell_hint_id":       shellhints.KindProbeShell,
}

// SyncTelemetryAfterShellHintBodyChange updates materialized shell fields on every telemetry row
// that references hintID, so config stays aligned with the hint body. Skips rows that fail validation.
func SyncTelemetryAfterShellHintBodyChange(ctx context.Context, st *store.Store, hintID uuid.UUID, kind, newBody string) (updated int, warnings []string) {
	idStr := hintID.String()
	rows, err := st.ListTelemetryReferencingShellHint(ctx, hintID)
	if err != nil {
		slog.Error("list telemetry for shell hint sync", "hint_id", idStr, "error", err)
		return 0, []string{fmt.Sprintf("could not list telemetry for sync: %v", err)}
	}
	for _, t := range rows {
		cfgMap, ok := cloneConfigJSONMap(t.ConfigJSON)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s (%s): config is not a JSON object", t.Name, t.ID))
			continue
		}
		target, ok := shellHintSyncTargetObject(cfgMap, t.Adapter)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s (%s): unsupported adapter or missing config branch for liveness", t.Name, t.ID))
			continue
		}
		slot := findShellHintSlot(target, idStr)
		if slot == "" {
			continue
		}
		slotKind, known := shellHintSlotKinds[slot]
		if !known {
			continue
		}
		if slotKind != kind {
			warnings = append(warnings, fmt.Sprintf("%s (%s): hint slot %s expects kind %s but hint is %s — skipped", t.Name, t.ID, slot, slotKind, kind))
			continue
		}
		if err := applyShellHintBodyToConfigObject(target, kind, newBody); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s (%s): %v", t.Name, t.ID, err))
			continue
		}

		cfgBytes, err := json.Marshal(cfgMap)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s (%s): marshal config: %v", t.Name, t.ID, err))
			continue
		}
		qosBytes, err := json.Marshal(t.QosThresholds)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s (%s): marshal qos: %v", t.Name, t.ID, err))
			continue
		}
		if len(qosBytes) == 0 || string(qosBytes) == "null" {
			qosBytes = []byte("{}")
		}
		execTarget := t.ExecutionTarget
		if execTarget == "" {
			execTarget = EffectiveExecutionTarget(t.Adapter, "")
		}
		if ve := ValidateTelemetryConfig(t.Adapter, cfgBytes, qosBytes, execTarget); len(ve) > 0 {
			warnings = append(warnings, fmt.Sprintf("%s (%s): invalid after sync: %s", t.Name, t.ID, strings.Join(ve, "; ")))
			continue
		}
		if he := ValidateTelemetryShellHintRefs(ctx, st, t.Adapter, cfgBytes); len(he) > 0 {
			warnings = append(warnings, fmt.Sprintf("%s (%s): hint refs invalid after sync: %s", t.Name, t.ID, strings.Join(he, "; ")))
			continue
		}
		in := store.CreateTelemetryInput{
			Name:            t.Name,
			DisplayName:     t.DisplayName,
			Adapter:         t.Adapter,
			ConfigJSON:      cfgMap,
			QosThresholds:   t.QosThresholds,
			ExecutionTarget: execTarget,
			SecretRef:       t.SecretRef,
			TimeoutMs:       t.TimeoutMs,
			Retries:         t.Retries,
		}
		if _, err := st.UpdateTelemetry(ctx, t.ID, in); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s (%s): save failed: %v", t.Name, t.ID, err))
			continue
		}
		updated++
	}
	if updated > 0 || len(warnings) > 0 {
		slog.Info("shell hint body sync", "hint_id", idStr, "kind", kind, "updated", updated, "warnings", len(warnings))
	}
	return updated, warnings
}

func cloneConfigJSONMap(cfg any) (map[string]any, bool) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}

// shellHintSyncTargetObject returns the JSON object that holds shell hint ids and materialized fields.
func shellHintSyncTargetObject(cfg map[string]any, adapter string) (obj map[string]any, ok bool) {
	switch adapter {
	case "kubernetes", "cli", "hlctl":
		return cfg, true
	case "liveness":
		raw, exists := cfg["kubernetes"]
		if !exists || raw == nil {
			return nil, false
		}
		nest, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		return nest, true
	default:
		return nil, false
	}
}

func findShellHintSlot(obj map[string]any, hintID string) string {
	for _, key := range []string{"job_env_hint_id", "job_node_selector_hint_id", "container_prep_hint_id", "probe_shell_hint_id"} {
		v, ok := obj[key]
		if !ok || v == nil {
			continue
		}
		s, ok := v.(string)
		if ok && strings.TrimSpace(s) == hintID {
			return key
		}
	}
	return ""
}

func applyShellHintBodyToConfigObject(obj map[string]any, kind, newBody string) error {
	switch kind {
	case shellhints.KindJobEnv:
		m, err := shellhints.ParseProbeEnvLines(newBody)
		if err != nil {
			return err
		}
		if len(m) == 0 {
			delete(obj, "env")
		} else {
			obj["env"] = m
		}
	case shellhints.KindJobNodeSelector:
		m, err := shellhints.ParseNodeSelectorLines(newBody)
		if err != nil {
			return err
		}
		if len(m) == 0 {
			delete(obj, "node_selector")
		} else {
			obj["node_selector"] = m
		}
	case shellhints.KindContainerPrep:
		obj["container_prep"] = strings.TrimSpace(newBody)
	case shellhints.KindProbeShell:
		obj["cli_shell"] = strings.TrimSpace(newBody)
	default:
		return fmt.Errorf("unknown hint kind %q", kind)
	}
	return nil
}
