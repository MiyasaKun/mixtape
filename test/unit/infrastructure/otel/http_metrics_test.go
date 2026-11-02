package otel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"fiber_test/internal/infrastructure/otel"
)

// TestHTTPMetricsExportsOnShutdown runs the HTTP traffic recorder against a
// real OTLP endpoint and verifies data is pushed out on shutdown.
func TestHTTPMetricsExportsOnShutdown(t *testing.T) {
	paths := make(chan string, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case paths <- path.Clean(r.URL.Path):
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setup, err := otel.Setup(context.Background(), otel.Config{
		Endpoint:    server.URL,
		ServiceName: "mixtape-test",
		Env:         "test",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	hm := setup.HTTPMetrics()
	if hm == nil {
		t.Fatal("HTTPMetrics() returned nil")
	}
	hm.ObserveRequest("GET", "/api/tracks/:id", 200, 0.25)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := setup.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	sawMetrics := false
	for drained := 0; ; drained++ {
		select {
		case p := <-paths:
			if p == "/v1/metrics" {
				sawMetrics = true
			}
		default:
			if drained == 0 {
				t.Fatalf("no OTLP requests reached the endpoint")
			}
			goto done
		}
	}
done:
	if !sawMetrics {
		t.Fatal("HTTP metrics were not exported to the endpoint")
	}
}
