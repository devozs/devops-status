package providerverify

import (
	"context"

	"github.com/devops-status/be/internal/adapter"
)

// VerifyPrometheus runs a minimal Prometheus API check (delegates to adapter).
func VerifyPrometheus(ctx context.Context, cfg adapter.PrometheusConfig) (latencyMs int, err error) {
	return adapter.VerifyPrometheusIntegration(ctx, cfg)
}
