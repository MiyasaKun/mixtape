// Package otel wires the OpenTelemetry SDK (metrics and logs) into the
// application. Telemetry is opt-in: when no OTLP endpoint is configured the
// setup is disabled and hands out no-op sinks, so the composition root never
// needs to special-case the disabled path.
package otel

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"fiber_test/internal/application/ports"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Config controls the OpenTelemetry SDK setup.
type Config struct {
	// Endpoint is the base OTLP HTTP endpoint (e.g.
	// http://otel-collector:4318). The signal-specific paths /v1/metrics and
	// /v1/logs are appended internally. An empty endpoint disables
	// telemetry entirely.
	Endpoint string

	// ServiceName identifies the service in the telemetry backend.
	ServiceName string

	// Env is recorded as the deployment environment attribute.
	Env string
}

// Telemetry holds the wired SDK for one application process.
type Telemetry struct {
	// Enabled reports whether telemetry was configured. It is false when no
	// endpoint was supplied.
	Enabled bool

	// LogHandler is an slog.Handler that exports log records via OTLP. It is
	// nil when telemetry is disabled.
	LogHandler slog.Handler

	loggerProvider *logsdk.LoggerProvider
	meterProvider  *metric.MeterProvider
}

// Setup wires the OTLP exporters and SDK providers, and returns a Setup that
// must be shut down on process exit. Both signals share one OTLP HTTP
// endpoint. Failed instrument setup is never fatal.
func Setup(ctx context.Context, cfg Config) (*Telemetry, error) {
	if cfg.Endpoint == "" {
		return &Telemetry{}, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironmentNameKey.String(cfg.Env),
		),
	)
	if err != nil {
		return nil, errors.Join(errors.New("otel: building resource"), err)
	}

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(strings.TrimSuffix(cfg.Endpoint, "/")+"/v1/metrics"))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating metric exporter"), err)
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(30*time.Second))),
	)

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(strings.TrimSuffix(cfg.Endpoint, "/")+"/v1/logs"))
	if err != nil {
		_ = meterProvider.Shutdown(ctx)
		return nil, errors.Join(errors.New("otel: creating log exporter"), err)
	}
	loggerProvider := logsdk.NewLoggerProvider(
		logsdk.WithResource(res),
		logsdk.WithProcessor(logsdk.NewBatchProcessor(logExporter, logsdk.WithExportInterval(30*time.Second))),
	)

	return &Telemetry{
		Enabled:        true,
		LogHandler:     otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider)),
		loggerProvider: loggerProvider,
		meterProvider:  meterProvider,
	}, nil
}

// Shutdown flushes all pending metrics and log records and stops the SDK.
// Shutdown on a disabled setup is a no-op.
func (s *Telemetry) Shutdown(ctx context.Context) error {
	if !s.Enabled {
		return nil
	}

	var err error
	if metricErr := s.meterProvider.Shutdown(ctx); metricErr != nil {
		err = errors.Join(err, errors.New("otel: shutting down meter provider: "+metricErr.Error()))
	}
	if logErr := s.loggerProvider.Shutdown(ctx); logErr != nil {
		err = errors.Join(err, errors.New("otel: shutting down logger provider: "+logErr.Error()))
	}
	return err
}

// Metrics returns a ports.Metrics backed by the wired SDK. A disabled or
// failed telemetry setup returns a NoopMetrics so callers never nil-check.
func (s *Telemetry) Metrics() ports.Metrics {
	if !s.Enabled || s.meterProvider == nil {
		return ports.NewNoopMetrics()
	}
	recorder, err := newMetricsRecorder(s.meterProvider)
	if err != nil {
		return ports.NewNoopMetrics()
	}
	return recorder
}
