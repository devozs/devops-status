package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProbeTestHandler struct {
	store            *store.Store
	secret           *secrets.Store
	k8sInsecureTLS   bool
}

func NewProbeTestHandler(s *store.Store, sec *secrets.Store, k8sInsecureTLS bool) *ProbeTestHandler {
	return &ProbeTestHandler{store: s, secret: sec, k8sInsecureTLS: k8sInsecureTLS}
}

type probeTestRequest struct {
	Adapter         string            `json:"adapter"`
	ConfigJSON      any               `json:"config_json"`
	Credentials     map[string]string `json:"credentials,omitempty"`
	TelemetryID     *uuid.UUID        `json:"telemetry_id,omitempty"`
	EnvironmentID   *uuid.UUID        `json:"environment_id,omitempty"`
	TestConnectOnly bool              `json:"test_connect_only,omitempty"`
	QosThresholds   any               `json:"qos_thresholds,omitempty"`
	ExecutionTarget string            `json:"execution_target,omitempty"`
}

func (h *ProbeTestHandler) Test(w http.ResponseWriter, r *http.Request) {
	var req probeTestRequest
	if err := handler.DecodeJSON(r, &req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	p, err := h.prepareProbeTest(r.Context(), &req)
	if err != nil {
		var mergeErr *mergeK8sProbeConfigError
		if errors.As(err, &mergeErr) {
			handler.WriteError(w, http.StatusInternalServerError, "failed to merge k8s config")
			return
		}
		var br *probeTestBadRequestErr
		if errors.As(err, &br) {
			handler.WriteError(w, http.StatusBadRequest, br.msg)
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	qosBytes := p.QosBytes

	if req.Adapter == "prometheus" && req.TestConnectOnly {
		var pc adapter.PrometheusConfig
		_ = json.Unmarshal(p.CfgBytes, &pc)
		if err := adapter.TestPrometheusConnectivity(r.Context(), pc); err != nil {
			handler.WriteJSON(w, http.StatusOK, map[string]any{
				"success":        false,
				"operational_ok": false,
				"error":          err.Error(),
			})
			return
		}
		handler.WriteJSON(w, http.StatusOK, map[string]any{
			"success":        true,
			"operational_ok": true,
		})
		return
	}

	ctx := r.Context()
	testCtx, cancel := context.WithTimeout(ctx, probeTestContextTimeout(req.Adapter, p.CfgBytes))
	defer cancel()

	result, err := h.executeProbeTest(testCtx, p)
	if err != nil {
		handler.WriteJSON(w, http.StatusOK, map[string]any{
			"success":        false,
			"operational_ok": false,
			"error":          err.Error(),
		})
		return
	}

	operationalOk := result.Success
	lvl := adapter.QoSLevelFromProbe(req.Adapter, qosBytes, result, operationalOk)

	out := map[string]any{
		"success":        result.Success,
		"operational_ok": operationalOk,
		"raw_value":      result.RawValue,
		"latency_ms":     result.LatencyMs,
		"metadata":       result.Metadata,
		"trace":          result.Trace,
		"error":          result.Error,
		"probed_at":      result.ProbedAt,
	}
	if req.Adapter == "cli" || req.Adapter == "hlctl" || (req.Adapter == "kubernetes" && p.ShellMetric) || (req.Adapter == "liveness" && p.LivenessShellMetric) {
		out["cli_logs"] = probeTestCLILogs(result.Metadata)
	}
	if lvl != "" {
		out["qos_level"] = lvl
	}
	handler.WriteJSON(w, http.StatusOK, out)
}

func probeTestCLILogs(meta map[string]any) map[string]any {
	if meta == nil {
		return map[string]any{}
	}
	out := make(map[string]any)
	for _, k := range []string{
		"cli_stdout", "cli_stderr", "cli_runner", "cli_image", "cli_log_combined", "cli_qos_attempts",
		"cli_text_value", "value_kind", "exit_code", "job", "namespace",
		"metric_value_source", "metric_green_band", "metric_yellow_band",
	} {
		if v, ok := meta[k]; ok {
			out[k] = v
		}
	}
	return out
}

func (h *ProbeTestHandler) applyEnvironmentBinding(ctx context.Context, req probeTestRequest, cfgBytes []byte) ([]byte, error) {
	if req.Adapter == "http" || req.Adapter == "prometheus" {
		return cfgBytes, nil
	}
	if req.Adapter == "hlctl" {
		return cfgBytes, nil
	}
	if req.Adapter == "liveness" {
		var cfg map[string]any
		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return cfgBytes, nil
		}
		src, _ := cfg["source"].(string)
		if strings.ToLower(strings.TrimSpace(src)) != "kubernetes" {
			return cfgBytes, nil
		}
		if req.EnvironmentID == nil {
			return cfgBytes, nil
		}
		cluster, err := h.store.GetConnectedClusterForEnvironment(ctx, *req.EnvironmentID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			return nil, fmt.Errorf("load cluster: %w", err)
		}
		nest, ok := cfg["kubernetes"].(map[string]any)
		if !ok || nest == nil {
			return nil, fmt.Errorf("liveness kubernetes config is required when source is kubernetes")
		}
		nest["cluster_id"] = cluster.ID.String()
		want, _ := nest["k8s_version"].(string)
		if want != "" && !k8sMinorVersionMatch(cluster.K8sVersion, want) {
			return nil, fmt.Errorf("cluster kubernetes version %s does not match telemetry target %s", cluster.K8sVersion, want)
		}
		cfg["kubernetes"] = nest
		return json.Marshal(cfg)
	}
	if req.EnvironmentID == nil {
		return cfgBytes, nil
	}
	cluster, err := h.store.GetConnectedClusterForEnvironment(ctx, *req.EnvironmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("load cluster: %w", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return cfgBytes, nil
	}

	switch req.Adapter {
	case "kubernetes":
		cfg["cluster_id"] = cluster.ID.String()
		want, _ := cfg["k8s_version"].(string)
		if want != "" && !k8sMinorVersionMatch(cluster.K8sVersion, want) {
			return nil, fmt.Errorf("cluster kubernetes version %s does not match telemetry target %s", cluster.K8sVersion, want)
		}
	case "cli":
		et := req.ExecutionTarget
		if et == "" {
			if v, ok := cfg["execution_target"].(string); ok {
				et = v
			}
		}
		if et == "k8s_cluster" {
			cfg["cluster_id"] = cluster.ID.String()
			want, _ := cfg["k8s_version"].(string)
			if want != "" && !k8sMinorVersionMatch(cluster.K8sVersion, want) {
				return nil, fmt.Errorf("cluster kubernetes version %s does not match telemetry target %s", cluster.K8sVersion, want)
			}
		}
	}

	return json.Marshal(cfg)
}

func k8sMinorVersionMatch(clusterVer, want string) bool {
	cv := strings.TrimPrefix(strings.TrimSpace(clusterVer), "v")
	parts := strings.Split(cv, ".")
	if len(parts) < 2 {
		return false
	}
	minor := parts[0] + "." + parts[1]
	return minor == strings.TrimSpace(want)
}
