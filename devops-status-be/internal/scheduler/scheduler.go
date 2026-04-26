package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/engine"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

type ProbeJob struct {
	TargetType    string
	TargetID      uuid.UUID
	TelemetryID   uuid.UUID
	Adapter       string
	ConfigJSON    json.RawMessage
	QosThresholds json.RawMessage
	IntervalSec   int
	WindowSize    int
	FailuresDown  int
	SuccessUp     int
}

type scheduleKey struct {
	TargetType  string
	TargetID    uuid.UUID
	TelemetryID uuid.UUID
}

type Scheduler struct {
	store          *store.Store
	secret         *secrets.Store
	evaluator      *engine.StatusEvaluator
	incidents      *engine.IncidentManager
	workerCount    int
	k8sInsecureTLS bool
	jobs           chan ProbeJob
	stopCh         chan struct{}
	wg             sync.WaitGroup
	ticker         *time.Ticker

	mu        sync.Mutex
	lastProbe map[scheduleKey]time.Time
}

func New(s *store.Store, sec *secrets.Store, eval *engine.StatusEvaluator, inc *engine.IncidentManager, workers int, k8sInsecureTLS bool) *Scheduler {
	if workers <= 0 {
		workers = 5
	}
	return &Scheduler{
		store:          s,
		secret:         sec,
		evaluator:      eval,
		incidents:      inc,
		workerCount:    workers,
		k8sInsecureTLS: k8sInsecureTLS,
		jobs:           make(chan ProbeJob, 200),
		stopCh:         make(chan struct{}),
		lastProbe:      make(map[scheduleKey]time.Time),
	}
}

func (sc *Scheduler) Start(ctx context.Context) {
	for i := 0; i < sc.workerCount; i++ {
		sc.wg.Add(1)
		go sc.worker(ctx, i)
	}

	sc.ticker = time.NewTicker(5 * time.Second)
	sc.wg.Add(1)
	go sc.scheduler(ctx)

	slog.Info("scheduler started", "workers", sc.workerCount)
}

func (sc *Scheduler) Stop() {
	close(sc.stopCh)
	if sc.ticker != nil {
		sc.ticker.Stop()
	}
	sc.wg.Wait()
	slog.Info("scheduler stopped")
}

func (sc *Scheduler) scheduler(ctx context.Context) {
	defer sc.wg.Done()

	sc.loadAndDispatch(ctx)

	for {
		select {
		case <-sc.stopCh:
			return
		case <-ctx.Done():
			return
		case <-sc.ticker.C:
			sc.loadAndDispatch(ctx)
		}
	}
}

func (sc *Scheduler) isDue(key scheduleKey, intervalSec int) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	last, ok := sc.lastProbe[key]
	if !ok {
		return true
	}
	interval := time.Duration(intervalSec) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return time.Since(last) >= interval
}

func (sc *Scheduler) markProbed(key scheduleKey) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.lastProbe[key] = time.Now()
}

func (sc *Scheduler) loadAndDispatch(ctx context.Context) {
	svcLinks, err := sc.store.ListServiceTelemetryLinks(ctx, nil)
	if err != nil {
		slog.Error("load service telemetry links", "error", err)
	}
	for _, b := range svcLinks {
		key := scheduleKey{TargetType: "service", TargetID: b.ServiceID, TelemetryID: b.TelemetryID}
		if !sc.isDue(key, b.SampleIntervalSec) {
			continue
		}
		t, err := sc.store.GetTelemetryByID(ctx, b.TelemetryID)
		if err != nil {
			slog.Error("load telemetry for service link", "link_id", b.ID, "error", err)
			continue
		}
		cfgBytes, _ := json.Marshal(t.ConfigJSON)
		qosBytes, _ := json.Marshal(t.QosThresholds)
		select {
		case sc.jobs <- ProbeJob{
			TargetType:    "service",
			TargetID:      b.ServiceID,
			TelemetryID:   b.TelemetryID,
			Adapter:       t.Adapter,
			ConfigJSON:    cfgBytes,
			QosThresholds: qosBytes,
			IntervalSec:   b.SampleIntervalSec,
			WindowSize:    b.WindowSize,
			FailuresDown:  b.ConsecutiveFailuresToDown,
			SuccessUp:     b.ConsecutiveSuccessToRecover,
		}:
			sc.markProbed(key)
		default:
			slog.Warn("job queue full, skipping", "target", b.ServiceID)
		}
	}

	envLinks, err := sc.store.ListEnvironmentTelemetryLinks(ctx, nil)
	if err != nil {
		slog.Error("load environment telemetry links", "error", err)
	}
	for _, b := range envLinks {
		key := scheduleKey{TargetType: "environment", TargetID: b.EnvironmentID, TelemetryID: b.TelemetryID}
		if !sc.isDue(key, b.SampleIntervalSec) {
			continue
		}
		t, err := sc.store.GetTelemetryByID(ctx, b.TelemetryID)
		if err != nil {
			slog.Error("load telemetry for environment link", "link_id", b.ID, "error", err)
			continue
		}
		cfgBytes, _ := json.Marshal(t.ConfigJSON)
		qosBytes, _ := json.Marshal(t.QosThresholds)
		select {
		case sc.jobs <- ProbeJob{
			TargetType:    "environment",
			TargetID:      b.EnvironmentID,
			TelemetryID:   b.TelemetryID,
			Adapter:       t.Adapter,
			ConfigJSON:    cfgBytes,
			QosThresholds: qosBytes,
			IntervalSec:   b.SampleIntervalSec,
			WindowSize:    b.WindowSize,
			FailuresDown:  b.ConsecutiveFailuresToDown,
			SuccessUp:     b.ConsecutiveSuccessToRecover,
		}:
			sc.markProbed(key)
		default:
			slog.Warn("job queue full, skipping", "target", b.EnvironmentID)
		}
	}
}

func (sc *Scheduler) worker(ctx context.Context, id int) {
	defer sc.wg.Done()
	for {
		select {
		case <-sc.stopCh:
			return
		case <-ctx.Done():
			return
		case job := <-sc.jobs:
			sc.executeProbe(ctx, job)
		}
	}
}

func cloneMetadata(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func hasQoSBytes(q []byte) bool {
	return len(q) > 0 && string(q) != "{}" && string(q) != "null"
}

func (sc *Scheduler) executeProbe(ctx context.Context, job ProbeJob) {
	a, err := adapter.Get(job.Adapter)
	if err != nil {
		slog.Error("adapter not found", "adapter", job.Adapter, "error", err)
		return
	}

	cfgJSON, err := sc.store.MergeKubernetesProbeConfig(ctx, job.Adapter, job.ConfigJSON, sc.k8sInsecureTLS)
	if err != nil {
		slog.Error("merge k8s credentials", "error", err)
		return
	}

	cfgJSON, err = sc.store.ResolveTelemetryProbeConfig(ctx, sc.secret, job.Adapter, cfgJSON)
	if err != nil {
		slog.Error("resolve telemetry probe config", "error", err)
		return
	}

	probeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	var result *adapter.ProbeResult
	shellMetric := adapter.HasCLIMetricQoS(cfgJSON, job.QosThresholds)
	var livenessShellMetric bool
	var livenessInner json.RawMessage
	var livenessSynQos []byte
	if job.Adapter == "liveness" {
		var w struct {
			LivenessCheck string          `json:"liveness_check"`
			Config        json.RawMessage `json:"config"`
		}
		if json.Unmarshal(cfgJSON, &w) == nil && strings.EqualFold(strings.TrimSpace(w.LivenessCheck), "kubernetes") {
			var kc adapter.KubernetesConfig
			if json.Unmarshal(w.Config, &kc) == nil && strings.TrimSpace(kc.CLIShell) != "" {
				if syn, ok := adapter.BuildSyntheticMetricQoSFromLivenessValue(job.QosThresholds); ok && adapter.HasCLIMetricQoS(w.Config, syn) {
					livenessShellMetric = true
					livenessInner = w.Config
					livenessSynQos = syn
				}
			}
		}
	}

	if (job.Adapter == "cli" || job.Adapter == "kubernetes" || job.Adapter == "hlctl") && shellMetric {
		result, err = adapter.RunCLIMetricShellProbe(probeCtx, a, cfgJSON, job.QosThresholds)
	} else if job.Adapter == "liveness" && livenessShellMetric {
		ka, kerr := adapter.Get("kubernetes")
		if kerr != nil {
			err = kerr
		} else {
			result, err = adapter.RunCLIMetricShellProbe(probeCtx, ka, livenessInner, livenessSynQos)
		}
	} else if (job.Adapter == "cli" || job.Adapter == "hlctl") && adapter.HasCLIPQoSThresholds(job.QosThresholds) {
		result, err = adapter.RunCLIPQoSProbe(probeCtx, a, cfgJSON, job.QosThresholds)
	}
	if result == nil {
		result, err = a.Probe(probeCtx, cfgJSON)
	}
	if err != nil {
		slog.Error("probe execution error", "adapter", job.Adapter, "target", job.TargetID, "error", err)
		result = &adapter.ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    err.Error(),
		}
	}

	var rawVal *float64
	qosForNumberKind := job.QosThresholds
	if job.Adapter == "liveness" && livenessShellMetric {
		qosForNumberKind = livenessSynQos
	}
	if (job.Adapter == "cli" || job.Adapter == "kubernetes" || job.Adapter == "hlctl") && shellMetric && adapter.CLIMetricValueKind(job.QosThresholds) == "number" {
		if result.Success {
			v := result.RawValue
			rawVal = &v
		}
	} else if job.Adapter == "liveness" && livenessShellMetric && adapter.CLIMetricValueKind(qosForNumberKind) == "number" {
		if result.Success {
			v := result.RawValue
			rawVal = &v
		}
	} else if result.RawValue != 0 {
		rawVal = &result.RawValue
	}
	latency := &result.LatencyMs

	tid := job.TelemetryID
	opRec := store.SampleRecord{
		TargetType:   job.TargetType,
		TargetID:     job.TargetID,
		TelemetryID:  &tid,
		ProbeKind:    "operational",
		SampledAt:    result.ProbedAt,
		Success:      result.Success,
		RawValue:     rawVal,
		LatencyMs:    latency,
		MetadataJSON: result.Metadata,
		SourceTrace:  result.Trace,
	}

	if job.Adapter == "cli" || (job.Adapter == "kubernetes" && shellMetric) || (job.Adapter == "liveness" && livenessShellMetric) {
		opRec.MetadataJSON = adapter.TrimCLIProbeMetadata(opRec.MetadataJSON)
	}
	if err := sc.store.InsertSampleResult(ctx, opRec); err != nil {
		slog.Error("store operational sample", "error", err)
		return
	}

	if hasQoSBytes(job.QosThresholds) && result.Success {
		meta := cloneMetadata(result.Metadata)
		if lvl := adapter.QoSLevelFromProbe(job.Adapter, job.QosThresholds, result, true); lvl != "" {
			meta["qos_level"] = lvl
		}
		qosRec := store.SampleRecord{
			TargetType:   job.TargetType,
			TargetID:     job.TargetID,
			TelemetryID:  &tid,
			ProbeKind:    "qos",
			SampledAt:    result.ProbedAt,
			Success:      true,
			RawValue:     rawVal,
			LatencyMs:    latency,
			MetadataJSON: meta,
			SourceTrace:  result.Trace,
		}
		if job.Adapter == "cli" || (job.Adapter == "kubernetes" && shellMetric) || (job.Adapter == "liveness" && livenessShellMetric) {
			qosRec.MetadataJSON = adapter.TrimCLIProbeMetadata(qosRec.MetadataJSON)
		}
		if err := sc.store.InsertSampleResult(ctx, qosRec); err != nil {
			slog.Error("store qos sample", "error", err)
			return
		}
	}

	sc.evaluator.EvaluateTargetAfterProbe(ctx, job.TargetType, job.TargetID)
}
