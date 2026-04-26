package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// mergeK8sProbeConfigError wraps store MergeKubernetesProbeConfig failures (HTTP 500).
type mergeK8sProbeConfigError struct {
	cause error
}

func (e *mergeK8sProbeConfigError) Error() string { return "failed to merge k8s config" }

func (e *mergeK8sProbeConfigError) Unwrap() error { return e.cause }

// probeTestPrepared holds resolved config and routing flags for probe test execution.
type probeTestPrepared struct {
	Req                 probeTestRequest
	Adapter             adapter.Adapter
	CfgBytes            []byte
	QosBytes            []byte
	ShellMetric         bool
	LivenessShellMetric bool
	LivenessInner       json.RawMessage
	LivenessSynQos      []byte
}

func (h *ProbeTestHandler) prepareProbeTest(ctx context.Context, req *probeTestRequest) (*probeTestPrepared, error) {
	a, err := adapter.Get(req.Adapter)
	if err != nil {
		return nil, errProbeTestBadRequest("unsupported adapter: " + req.Adapter)
	}

	cfgBytes, err := json.Marshal(req.ConfigJSON)
	if err != nil {
		return nil, errProbeTestBadRequest("invalid config_json")
	}

	cfgBytes, err = h.applyEnvironmentBinding(ctx, *req, cfgBytes)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, pgx.ErrNoRows) {
			msg = "no connected kubernetes cluster for this environment"
		}
		return nil, errProbeTestBadRequest(msg)
	}

	cfgBytes, err = h.store.MergeKubernetesProbeConfig(ctx, req.Adapter, cfgBytes, h.k8sInsecureTLS)
	if err != nil {
		return nil, &mergeK8sProbeConfigError{cause: err}
	}

	cfgBytes, err = h.store.ResolveTelemetryProbeConfig(ctx, h.secret, req.Adapter, cfgBytes)
	if err != nil {
		return nil, errProbeTestBadRequest(err.Error())
	}

	qosBytes, _ := json.Marshal(req.QosThresholds)
	shellMetric := adapter.HasCLIMetricQoS(cfgBytes, qosBytes)
	var livenessShellMetric bool
	var livenessInner json.RawMessage
	var livenessSynQos []byte
	if req.Adapter == "liveness" {
		var w struct {
			LivenessCheck string          `json:"liveness_check"`
			Config        json.RawMessage `json:"config"`
		}
		if json.Unmarshal(cfgBytes, &w) == nil && strings.EqualFold(strings.TrimSpace(w.LivenessCheck), "kubernetes") {
			var kc adapter.KubernetesConfig
			if json.Unmarshal(w.Config, &kc) == nil && strings.TrimSpace(kc.CLIShell) != "" {
				if syn, ok := adapter.BuildSyntheticMetricQoSFromLivenessValue(qosBytes); ok && adapter.HasCLIMetricQoS(w.Config, syn) {
					livenessShellMetric = true
					livenessInner = w.Config
					livenessSynQos = syn
				}
			}
		}
	}

	return &probeTestPrepared{
		Req:                 *req,
		Adapter:             a,
		CfgBytes:            cfgBytes,
		QosBytes:            qosBytes,
		ShellMetric:         shellMetric,
		LivenessShellMetric: livenessShellMetric,
		LivenessInner:       livenessInner,
		LivenessSynQos:      livenessSynQos,
	}, nil
}

type probeTestBadRequestErr struct{ msg string }

func (e *probeTestBadRequestErr) Error() string { return e.msg }

func errProbeTestBadRequest(msg string) error { return &probeTestBadRequestErr{msg: msg} }

func timeoutMsFromAny(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}

// extractProbeConfigTimeoutMs reads timeout_ms from resolved config (HLCTL/CLI/K8s shell use top level; liveness+k8s uses kubernetes.timeout_ms).
func extractProbeConfigTimeoutMs(adapter string, cfgBytes []byte) int {
	var root map[string]any
	if json.Unmarshal(cfgBytes, &root) != nil {
		return 0
	}
	if adapter == "liveness" {
		if nest, ok := root["kubernetes"].(map[string]any); ok && nest != nil {
			if v, ok := nest["timeout_ms"]; ok {
				return timeoutMsFromAny(v)
			}
		}
	}
	if v, ok := root["timeout_ms"]; ok {
		return timeoutMsFromAny(v)
	}
	return 0
}

// probeTestContextTimeout bounds how long Verify may run. Uses config timeout_ms + buffer when set (e.g. HLCTL for several minutes); otherwise 90s.
func probeTestContextTimeout(adapter string, cfgBytes []byte) time.Duration {
	const defaultProbeTest = 90 * time.Second
	const maxProbeTest = 45 * time.Minute
	const buffer = 3 * time.Minute

	ms := extractProbeConfigTimeoutMs(adapter, cfgBytes)
	if ms <= 0 {
		return defaultProbeTest
	}
	d := time.Duration(ms)*time.Millisecond + buffer
	if d < defaultProbeTest {
		return defaultProbeTest
	}
	if d > maxProbeTest {
		return maxProbeTest
	}
	return d
}

func (h *ProbeTestHandler) executeProbeTest(ctx context.Context, p *probeTestPrepared) (*adapter.ProbeResult, error) {
	req := p.Req
	var result *adapter.ProbeResult
	var err error

	if (req.Adapter == "cli" || req.Adapter == "kubernetes" || req.Adapter == "hlctl") && p.ShellMetric {
		result, err = adapter.RunCLIMetricShellProbe(ctx, p.Adapter, p.CfgBytes, p.QosBytes)
	} else if req.Adapter == "liveness" && p.LivenessShellMetric {
		ka, kerr := adapter.Get("kubernetes")
		if kerr != nil {
			err = kerr
		} else {
			result, err = adapter.RunCLIMetricShellProbe(ctx, ka, p.LivenessInner, p.LivenessSynQos)
		}
	} else if (req.Adapter == "cli" || req.Adapter == "hlctl") && adapter.HasCLIPQoSThresholds(p.QosBytes) {
		result, err = adapter.RunCLIPQoSProbe(ctx, p.Adapter, p.CfgBytes, p.QosBytes)
	} else {
		result, err = p.Adapter.Probe(ctx, p.CfgBytes)
	}
	return result, err
}

func (h *ProbeTestHandler) probeHostMetadata(ctx context.Context, adapterName string, cfgBytes []byte) map[string]any {
	out := map[string]any{}
	var cid string
	switch adapterName {
	case "kubernetes":
		var cfg map[string]any
		if json.Unmarshal(cfgBytes, &cfg) == nil {
			if v, ok := cfg["cluster_id"].(string); ok {
				cid = strings.TrimSpace(v)
			}
		}
	case "cli":
		var cfg map[string]any
		if json.Unmarshal(cfgBytes, &cfg) == nil {
			if v, ok := cfg["cluster_id"].(string); ok {
				cid = strings.TrimSpace(v)
			}
		}
	case "liveness":
		var cfg map[string]any
		if json.Unmarshal(cfgBytes, &cfg) == nil {
			if nest, ok := cfg["kubernetes"].(map[string]any); ok && nest != nil {
				if v, ok := nest["cluster_id"].(string); ok {
					cid = strings.TrimSpace(v)
				}
			}
		}
	}
	if cid == "" {
		return out
	}
	out["cluster_id"] = cid
	id, err := uuid.Parse(cid)
	if err != nil {
		return out
	}
	cl, err := h.store.GetK8sClusterByID(ctx, id)
	if err != nil {
		return out
	}
	out["cluster_name"] = cl.Name
	return out
}
