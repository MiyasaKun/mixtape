package services

import (
	"fiber_test/internal/application/ports"
	"fiber_test/internal/infrastructure/logging"
)

// Telemetry holds the optional observational collaborators a service is built
// with: a structured logger and a metrics recorder. Services take it as a
// variadic trailing option, so existing call sites (and tests) work without
// telemetry, and the zero value resolves to safe no-ops.
type Telemetry struct {
	Logger  *logging.Logger
	Metrics ports.Metrics
}

type telemetryDeps struct {
	logger  *logging.Logger
	metrics ports.Metrics
}

// resolveTelemetry normalizes the variadic option into concrete, never-nil
// collaborators: a discard logger and a metrics recorder that drops
// everything.
func resolveTelemetry(opts []Telemetry) telemetryDeps {
	deps := telemetryDeps{
		logger:  logging.NewNoopLogger(),
		metrics: ports.NewNoopMetrics(),
	}
	if len(opts) == 0 {
		return deps
	}
	if opts[0].Logger != nil {
		deps.logger = opts[0].Logger
	}
	if opts[0].Metrics != nil {
		deps.metrics = opts[0].Metrics
	}
	return deps
}
