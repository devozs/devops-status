package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/handler"
)

type sseProbeSink struct {
	mu              sync.Mutex
	w               http.ResponseWriter
	rc              *http.ResponseController
	seq             int
	closed          bool
	writeSlack      time.Duration // rolling SetWriteDeadline window (matches probe budget)
}

func newSSEProbeSink(w http.ResponseWriter, rc *http.ResponseController, writeSlack time.Duration) *sseProbeSink {
	if writeSlack < 2*time.Minute {
		writeSlack = 2 * time.Minute
	}
	return &sseProbeSink{w: w, rc: rc, writeSlack: writeSlack}
}

func flushStream(rc *http.ResponseController) {
	if rc == nil {
		return
	}
	_ = rc.Flush()
}

func (s *sseProbeSink) bumpDeadline() {
	if s.rc != nil {
		_ = s.rc.SetWriteDeadline(time.Now().Add(s.writeSlack))
	}
}

func (s *sseProbeSink) WriteHost(meta map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.bumpDeadline()
	s.seq++
	payload := map[string]any{"meta": meta, "seq": s.seq}
	s.writeEventLocked("host", payload)
	flushStream(s.rc)
}

func (s *sseProbeSink) WriteLog(stream string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.bumpDeadline()
	s.seq++
	payload := map[string]any{
		"stream": stream,
		"chunk":  base64.StdEncoding.EncodeToString(data),
		"seq":    s.seq,
	}
	s.writeEventLocked("log", payload)
	flushStream(s.rc)
}

func (s *sseProbeSink) Heartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.bumpDeadline()
	s.writeEventLocked("heartbeat", map[string]any{"ts": time.Now().UTC().Format(time.RFC3339Nano)})
	flushStream(s.rc)
}

func (s *sseProbeSink) writeEventLocked(event string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, string(b))
}

func (s *sseProbeSink) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// TestStream runs the same probe as Test but streams shell output via Server-Sent Events (POST body identical to /probes/test).
func (h *ProbeTestHandler) TestStream(w http.ResponseWriter, r *http.Request) {
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

	rc := http.NewResponseController(w)
	streamBudget := probeTestContextTimeout(p.Req.Adapter, p.CfgBytes)
	if streamBudget < 2*time.Minute {
		streamBudget = 2 * time.Minute
	}
	_ = rc.SetWriteDeadline(time.Now().Add(streamBudget))

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sink := newSSEProbeSink(w, rc, streamBudget)

	host := h.probeHostMetadata(r.Context(), p.Req.Adapter, p.CfgBytes)
	enrichProbeStreamHost(p.Req.Adapter, p.CfgBytes, host)
	sink.WriteHost(host)

	hbDone := make(chan struct{})
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-hbDone:
				return
			case <-t.C:
				sink.Heartbeat()
			}
		}
	}()
	defer close(hbDone)

	ctx := r.Context()
	testCtx, cancel := context.WithTimeout(ctx, probeTestContextTimeout(p.Req.Adapter, p.CfgBytes))
	defer cancel()

	if p.Req.Adapter == "prometheus" && p.Req.TestConnectOnly {
		var pc adapter.PrometheusConfig
		_ = json.Unmarshal(p.CfgBytes, &pc)
		if err := adapter.TestPrometheusConnectivity(testCtx, pc); err != nil {
			h.writeStreamDone(sink, w, rc, p, nil, err, nil)
			return
		}
		h.writeStreamDone(sink, w, rc, p, nil, nil, map[string]any{
			"success": true, "operational_ok": true,
		})
		return
	}

	probeCtx := adapter.WithProbeLogSink(testCtx, sink)
	result, execErr := h.executeProbeTest(probeCtx, p)
	h.writeStreamDone(sink, w, rc, p, result, execErr, nil)
}

func enrichProbeStreamHost(adapterName string, cfgBytes []byte, host map[string]any) {
	if host == nil {
		return
	}
	switch adapterName {
	case "cli":
		var cfg map[string]any
		if json.Unmarshal(cfgBytes, &cfg) == nil {
			if v, ok := cfg["execution_target"].(string); ok && strings.TrimSpace(v) != "" {
				host["execution_target"] = strings.TrimSpace(v)
			}
		}
		host["shell_host_kind"] = "cli"
	case "kubernetes":
		host["execution_target"] = "k8s_cluster"
		host["shell_host_kind"] = "kubernetes_shell"
	case "hlctl":
		host["execution_target"] = "hlctl_local"
		host["shell_host_kind"] = "hlctl"
	case "liveness":
		host["execution_target"] = "k8s_cluster"
		host["shell_host_kind"] = "liveness_kubernetes_shell"
	default:
		host["shell_host_kind"] = adapterName
	}
}

func (h *ProbeTestHandler) writeStreamDone(sink *sseProbeSink, w http.ResponseWriter, rc *http.ResponseController, p *probeTestPrepared, result *adapter.ProbeResult, execErr error, override map[string]any) {
	sink.Close()
	if rc != nil {
		_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
	}

	var out map[string]any
	if override != nil {
		out = override
	} else if execErr != nil {
		out = map[string]any{
			"success":        false,
			"operational_ok": false,
			"error":          execErr.Error(),
		}
	} else {
		req := p.Req
		operationalOk := result.Success
		lvl := adapter.QoSLevelFromProbe(req.Adapter, p.QosBytes, result, operationalOk)
		out = map[string]any{
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
	}

	b, err := json.Marshal(out)
	if err != nil {
		b = []byte(`{"success":false,"error":"encode result"}`)
	}
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(b))
	flushStream(rc)
}
