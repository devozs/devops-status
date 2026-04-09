package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/engine"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

type ProbeJob struct {
	TargetType   string
	TargetID     uuid.UUID
	ProbeKind    string
	DataSourceID uuid.UUID
	Adapter      string
	ConfigJSON   json.RawMessage
	IntervalSec  int
	WindowSize   int
	FailuresDown int
	SuccessUp    int
}

type bindingKey struct {
	TargetType string
	TargetID   uuid.UUID
	ProbeKind  string
}

type Scheduler struct {
	store       *store.Store
	evaluator   *engine.StatusEvaluator
	incidents   *engine.IncidentManager
	workerCount int
	jobs        chan ProbeJob
	stopCh      chan struct{}
	wg          sync.WaitGroup
	ticker      *time.Ticker

	mu        sync.Mutex
	lastProbe map[bindingKey]time.Time
}

func New(s *store.Store, eval *engine.StatusEvaluator, inc *engine.IncidentManager, workers int) *Scheduler {
	if workers <= 0 {
		workers = 5
	}
	return &Scheduler{
		store:       s,
		evaluator:   eval,
		incidents:   inc,
		workerCount: workers,
		jobs:        make(chan ProbeJob, 200),
		stopCh:      make(chan struct{}),
		lastProbe:   make(map[bindingKey]time.Time),
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

func (sc *Scheduler) isDue(key bindingKey, intervalSec int) bool {
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

func (sc *Scheduler) markProbed(key bindingKey) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.lastProbe[key] = time.Now()
}

func (sc *Scheduler) loadAndDispatch(ctx context.Context) {
	svcBindings, err := sc.store.ListServiceProbeBindings(ctx, nil)
	if err != nil {
		slog.Error("load service bindings", "error", err)
	}
	for _, b := range svcBindings {
		key := bindingKey{TargetType: "service", TargetID: b.ServiceID, ProbeKind: b.ProbeKind}
		if !sc.isDue(key, b.SampleIntervalSec) {
			continue
		}
		ds, err := sc.store.GetDataSourceByID(ctx, b.DataSourceID)
		if err != nil {
			slog.Error("load data source for binding", "binding_id", b.ID, "error", err)
			continue
		}
		cfgBytes, _ := json.Marshal(ds.ConfigJSON)
		select {
		case sc.jobs <- ProbeJob{
			TargetType:   "service",
			TargetID:     b.ServiceID,
			ProbeKind:    b.ProbeKind,
			DataSourceID: b.DataSourceID,
			Adapter:      ds.Adapter,
			ConfigJSON:   cfgBytes,
			IntervalSec:  b.SampleIntervalSec,
			WindowSize:   b.WindowSize,
			FailuresDown: b.ConsecutiveFailuresToDown,
			SuccessUp:    b.ConsecutiveSuccessToRecover,
		}:
			sc.markProbed(key)
		default:
			slog.Warn("job queue full, skipping", "target", b.ServiceID)
		}
	}

	envBindings, err := sc.store.ListEnvironmentProbeBindings(ctx, nil)
	if err != nil {
		slog.Error("load environment bindings", "error", err)
	}
	for _, b := range envBindings {
		key := bindingKey{TargetType: "environment", TargetID: b.EnvironmentID, ProbeKind: b.ProbeKind}
		if !sc.isDue(key, b.SampleIntervalSec) {
			continue
		}
		ds, err := sc.store.GetDataSourceByID(ctx, b.DataSourceID)
		if err != nil {
			slog.Error("load data source for binding", "binding_id", b.ID, "error", err)
			continue
		}
		cfgBytes, _ := json.Marshal(ds.ConfigJSON)
		select {
		case sc.jobs <- ProbeJob{
			TargetType:   "environment",
			TargetID:     b.EnvironmentID,
			ProbeKind:    b.ProbeKind,
			DataSourceID: b.DataSourceID,
			Adapter:      ds.Adapter,
			ConfigJSON:   cfgBytes,
			IntervalSec:  b.SampleIntervalSec,
			WindowSize:   b.WindowSize,
			FailuresDown: b.ConsecutiveFailuresToDown,
			SuccessUp:    b.ConsecutiveSuccessToRecover,
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

func (sc *Scheduler) executeProbe(ctx context.Context, job ProbeJob) {
	a, err := adapter.Get(job.Adapter)
	if err != nil {
		slog.Error("adapter not found", "adapter", job.Adapter, "error", err)
		return
	}

	probeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	result, err := a.Probe(probeCtx, job.ConfigJSON)
	if err != nil {
		slog.Error("probe execution error", "adapter", job.Adapter, "target", job.TargetID, "error", err)
		result = &adapter.ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    err.Error(),
		}
	}

	var rawVal *float64
	if result.RawValue != 0 {
		rawVal = &result.RawValue
	}
	latency := &result.LatencyMs

	rec := store.SampleRecord{
		TargetType:   job.TargetType,
		TargetID:     job.TargetID,
		ProbeKind:    job.ProbeKind,
		SampledAt:    result.ProbedAt,
		Success:      result.Success,
		RawValue:     rawVal,
		LatencyMs:    latency,
		MetadataJSON: result.Metadata,
		SourceTrace:  result.Trace,
	}

	if err := sc.store.InsertSampleResult(ctx, rec); err != nil {
		slog.Error("store sample result", "error", err)
		return
	}

	sc.evaluator.Evaluate(ctx, job.TargetType, job.TargetID, job.ProbeKind, job.WindowSize, job.FailuresDown, job.SuccessUp)
}
