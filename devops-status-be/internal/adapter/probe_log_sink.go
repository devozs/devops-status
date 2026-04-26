package adapter

import "context"

// ProbeLogSink receives incremental shell output during probe execution (e.g. SSE to the admin UI).
type ProbeLogSink interface {
	WriteHost(meta map[string]any)
	WriteLog(stream string, data []byte)
}

type ctxKeyProbeLogSink struct{}

// WithProbeLogSink attaches a log sink to ctx for RunCLIAsDocker / RunCLIAsK8sJob / RunHLCTLLocalProbe.
func WithProbeLogSink(ctx context.Context, sink ProbeLogSink) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyProbeLogSink{}, sink)
}

// ProbeLogSinkFromContext returns the sink if set.
func ProbeLogSinkFromContext(ctx context.Context) ProbeLogSink {
	s, _ := ctx.Value(ctxKeyProbeLogSink{}).(ProbeLogSink)
	return s
}
