package admin

import (
	"context"
	"encoding/json"
	"strings"

	adapt "github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/shellhints"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

// ValidateTelemetryShellHintRefs ensures optional *_hint_id fields reference existing hints of the right kind
// and that the materialized config matches the hint body (strict).
func ValidateTelemetryShellHintRefs(ctx context.Context, st *store.Store, adapter string, configJSON []byte) []string {
	var errs []string
	add := func(msg string) { errs = append(errs, msg) }

	switch adapter {
	case "kubernetes":
		var cfg adapt.KubernetesConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid Kubernetes config JSON for hint validation"}
		}
		shell := strings.TrimSpace(cfg.CLIShell) != ""
		validateK8sShellHintFields(ctx, st, add, shell, &cfg)
	case "cli":
		var cfg adapt.CLIConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid CLI config JSON for hint validation"}
		}
		validateCLIHintFields(ctx, st, add, &cfg)
	case "hlctl":
		var cfg adapt.CLIConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid HLCTL config JSON for hint validation"}
		}
		validateCLIHintFields(ctx, st, add, &cfg)
	case "liveness":
		var cfg struct {
			Source     string          `json:"source"`
			Kubernetes json.RawMessage `json:"kubernetes"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid liveness config JSON for hint validation"}
		}
		if strings.ToLower(strings.TrimSpace(cfg.Source)) != "kubernetes" {
			var k adapt.KubernetesConfig
			if len(cfg.Kubernetes) > 0 {
				_ = json.Unmarshal(cfg.Kubernetes, &k)
			}
			if hintAnySet(k) {
				add("liveness shell hint links are only valid when source is kubernetes")
			}
			break
		}
		var k adapt.KubernetesConfig
		if err := json.Unmarshal(cfg.Kubernetes, &k); err != nil {
			add("invalid liveness kubernetes JSON for hint validation")
			break
		}
		shell := strings.TrimSpace(k.CLIShell) != ""
		validateK8sShellHintFields(ctx, st, add, shell, &k, "liveness kubernetes: ")
	default:
	}
	return errs
}

func hintAnySet(k adapt.KubernetesConfig) bool {
	return strings.TrimSpace(k.JobEnvHintID) != "" ||
		strings.TrimSpace(k.JobNodeSelectorHintID) != "" ||
		strings.TrimSpace(k.ContainerPrepHintID) != "" ||
		strings.TrimSpace(k.ProbeShellHintID) != ""
}

func validateK8sShellHintFields(ctx context.Context, st *store.Store, add func(string), shell bool, cfg *adapt.KubernetesConfig, prefix ...string) {
	p := ""
	if len(prefix) > 0 {
		p = prefix[0]
	}
	if !shell {
		if hintAnySet(*cfg) {
			add(p + "shell hint links require kubernetes shell mode (non-empty cli_shell)")
		}
		return
	}
	checkMapHint(ctx, st, add, p, cfg.JobEnvHintID, shellhints.KindJobEnv, func(h *model.TelemetryShellHint) bool {
		env := cfg.Env
		if env == nil {
			env = map[string]string{}
		}
		parsed, err := shellhints.ParseProbeEnvLines(h.Body)
		if err != nil {
			return false
		}
		return shellhints.MapsEqual(env, parsed)
	})
	checkMapHint(ctx, st, add, p, cfg.JobNodeSelectorHintID, shellhints.KindJobNodeSelector, func(h *model.TelemetryShellHint) bool {
		sel := cfg.NodeSelector
		if sel == nil {
			sel = map[string]string{}
		}
		parsed, err := shellhints.ParseNodeSelectorLines(h.Body)
		if err != nil {
			return false
		}
		return shellhints.MapsEqual(sel, parsed)
	})
	checkStringHint(ctx, st, add, p, cfg.ContainerPrepHintID, shellhints.KindContainerPrep, cfg.ContainerPrep, true)
	checkStringHint(ctx, st, add, p, cfg.ProbeShellHintID, shellhints.KindProbeShell, cfg.CLIShell, false)
}

func validateCLIHintFields(ctx context.Context, st *store.Store, add func(string), cfg *adapt.CLIConfig) {
	checkMapHint(ctx, st, add, "", cfg.JobEnvHintID, shellhints.KindJobEnv, func(h *model.TelemetryShellHint) bool {
		env := cfg.Env
		if env == nil {
			env = map[string]string{}
		}
		parsed, err := shellhints.ParseProbeEnvLines(h.Body)
		if err != nil {
			return false
		}
		return shellhints.MapsEqual(env, parsed)
	})
	checkMapHint(ctx, st, add, "", cfg.JobNodeSelectorHintID, shellhints.KindJobNodeSelector, func(h *model.TelemetryShellHint) bool {
		sel := cfg.NodeSelector
		if sel == nil {
			sel = map[string]string{}
		}
		parsed, err := shellhints.ParseNodeSelectorLines(h.Body)
		if err != nil {
			return false
		}
		return shellhints.MapsEqual(sel, parsed)
	})
	checkStringHint(ctx, st, add, "", cfg.ContainerPrepHintID, shellhints.KindContainerPrep, cfg.ContainerPrep, true)
	shell := strings.TrimSpace(cfg.CLIShell)
	if strings.TrimSpace(cfg.ProbeShellHintID) != "" && shell == "" {
		add("probe_shell_hint_id requires non-empty cli_shell")
		return
	}
	checkStringHint(ctx, st, add, "", cfg.ProbeShellHintID, shellhints.KindProbeShell, cfg.CLIShell, false)
}

// checkMapHint validates UUID hint for map-backed fields (env, node_selector).
func checkMapHint(ctx context.Context, st *store.Store, add func(string), prefix, idStr, wantKind string, match func(*model.TelemetryShellHint) bool) {
	idStr = strings.TrimSpace(idStr)
	if idStr == "" {
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		add(prefix + wantKind + " hint id must be a valid UUID")
		return
	}
	h, err := st.GetTelemetryShellHintByID(ctx, id)
	if err != nil {
		add(prefix + "failed to resolve shell hint")
		return
	}
	if h == nil {
		add(prefix + wantKind + " hint not found")
		return
	}
	if h.Kind != wantKind {
		add(prefix + "shell hint kind mismatch for " + wantKind)
		return
	}
	if !match(h) {
		add(prefix + "telemetry content must match linked " + wantKind + " hint body")
	}
}

func checkStringHint(ctx context.Context, st *store.Store, add func(string), prefix, idStr, wantKind, field string, allowBothEmpty bool) {
	idStr = strings.TrimSpace(idStr)
	fieldTrim := strings.TrimSpace(field)
	if idStr == "" {
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		add(prefix + wantKind + " hint id must be a valid UUID")
		return
	}
	h, err := st.GetTelemetryShellHintByID(ctx, id)
	if err != nil {
		add(prefix + "failed to resolve shell hint")
		return
	}
	if h == nil {
		add(prefix + wantKind + " hint not found")
		return
	}
	if h.Kind != wantKind {
		add(prefix + "shell hint kind mismatch for " + wantKind)
		return
	}
	hTrim := strings.TrimSpace(h.Body)
	if allowBothEmpty && fieldTrim == "" && hTrim == "" {
		return
	}
	if fieldTrim != hTrim {
		add(prefix + "telemetry content must match linked " + wantKind + " hint body")
	}
}
