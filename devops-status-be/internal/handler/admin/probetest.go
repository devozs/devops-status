package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProbeTestHandler struct {
	store          *store.Store
	secret         *secrets.Store
	k8sInsecureTLS bool
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

	a, err := adapter.Get(req.Adapter)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "unsupported adapter: "+req.Adapter)
		return
	}

	cfgBytes, err := json.Marshal(req.ConfigJSON)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid config_json")
		return
	}

	cfgBytes, err = h.applyEnvironmentBinding(r.Context(), req, cfgBytes)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, pgx.ErrNoRows) {
			msg = "no connected kubernetes cluster for this environment"
		}
		handler.WriteError(w, http.StatusBadRequest, msg)
		return
	}

	cfgBytes, err = h.store.MergeKubernetesProbeConfig(r.Context(), req.Adapter, cfgBytes, h.k8sInsecureTLS)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to merge k8s config")
		return
	}

	cfgBytes, err = h.store.ResolveTelemetryProbeConfig(r.Context(), h.secret, req.Adapter, cfgBytes)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	qosBytes, _ := json.Marshal(req.QosThresholds)

	if req.Adapter == "prometheus" && req.TestConnectOnly {
		var pc adapter.PrometheusConfig
		_ = json.Unmarshal(cfgBytes, &pc)
		if err := adapter.TestPrometheusConnectivity(r.Context(), pc); err != nil {
			handler.WriteJSON(w, http.StatusOK, map[string]any{
				"success":         false,
				"operational_ok":  false,
				"error":           err.Error(),
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
	testCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	var result *adapter.ProbeResult
	if req.Adapter == "cli" && adapter.HasCLIMetricQoS(cfgBytes, qosBytes) {
		result, err = adapter.RunCLIMetricShellProbe(testCtx, a, cfgBytes, qosBytes)
	} else if req.Adapter == "cli" && adapter.HasCLIPQoSThresholds(qosBytes) {
		result, err = adapter.RunCLIPQoSProbe(testCtx, a, cfgBytes, qosBytes)
	} else {
		result, err = a.Probe(testCtx, cfgBytes)
	}
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
	if req.Adapter == "cli" {
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
	for _, k := range []string{"cli_stdout", "cli_stderr", "cli_runner", "cli_image", "cli_log_combined", "cli_qos_attempts", "cli_text_value", "value_kind", "exit_code", "job", "namespace"} {
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

