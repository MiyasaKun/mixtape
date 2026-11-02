package otel

import (
	"context"
	"errors"

	"fiber_test/internal/adapters/inbound/http/middleware"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// httpRecorder implements middleware.HTTPMetrics on top of the OTel SDK,
// recording request counts and latency histograms per method/route/status.
type httpRecorder struct {
	requests metric.Int64Counter
	latency  metric.Float64Histogram
}

// noopHTTPMetrics drops every observation; it is the fallback when
// telemetry is disabled.
type noopHTTPMetrics struct{}

func (noopHTTPMetrics) ObserveRequest(string, string, int, float64) {}

var _ middleware.HTTPMetrics = noopHTTPMetrics{}

// HTTPMetrics builds the HTTP traffic recorder. A disabled or failed
// telemetry setup returns the noop recorder so callers never nil-check.
func (s *Telemetry) HTTPMetrics() middleware.HTTPMetrics {
	if !s.Enabled || s.meterProvider == nil {
		return noopHTTPMetrics{}
	}
	if r, err := newHTTPRecorder(s.meterProvider); err == nil {
		return r
	}
	return noopHTTPMetrics{}
}

func newHTTPRecorder(mp *sdkmetric.MeterProvider) (middleware.HTTPMetrics, error) {
	meter := mp.Meter(scopeName)

	requests, err := meter.Int64Counter("http_server_requests_total",
		metric.WithDescription("HTTP requests, by method, route template and status code."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating http_server_requests_total"), err)
	}

	latency, err := meter.Float64Histogram("http_server_request_duration_seconds",
		metric.WithUnit("s"),
		metric.WithDescription("HTTP request duration, by method and route template."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating http_server_request_duration_seconds"), err)
	}

	return &httpRecorder{requests: requests, latency: latency}, nil
}

func httpAttrs(method, route string, status int) metric.MeasurementOption {
	return metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("route", route),
		attribute.Int("status", status),
	)
}

func (h *httpRecorder) ObserveRequest(method, route string, status int, seconds float64) {
	o := httpAttrs(method, route, status)
	h.requests.Add(context.Background(), 1, o)
	h.latency.Record(context.Background(), seconds, o)
}

var _ middleware.HTTPMetrics = (*httpRecorder)(nil)
