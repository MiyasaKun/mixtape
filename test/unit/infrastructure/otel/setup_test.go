package otel_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"fiber_test/internal/infrastructure/otel"
)

func TestSetupWithoutEndpointIsDisabled(t *testing.T) {
	setup, err := otel.Setup(context.Background(), otel.Config{
		Endpoint:    "",
		ServiceName: "mixtape",
		Env:         "test",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if setup.Enabled {
		t.Fatal("Setup() enabled without endpoint configured")
	}
	if err := setup.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	// Disabled setups must still hand out harmless telemetry sinks so callers
	// never need nil checks.
	setup.Metrics().CommandHandled("play", "ok")
	if setup.LogHandler != nil {
		t.Fatal("disabled setup should hand back no log handler")
	}
}

func TestSetupExportsMetricsAndLogsOnShutdown(t *testing.T) {
	seenCh := make(chan string, 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case seenCh <- path.Clean(r.URL.Path):
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
	if !setup.Enabled {
		t.Fatal("Setup() not enabled with endpoint configured")
	}

	setup.Metrics().CommandHandled("play", "ok")

	if setup.LogHandler == nil {
		t.Fatal("enabled setup returned nil log handler")
	}
	slog.New(setup.LogHandler).Info("test log record")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := setup.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	seen := map[string]int{}
	for drained := 0; ; drained++ {
		select {
		case p := <-seenCh:
			seen[p]++
		default:
			if drained == 0 {
				t.Fatalf("no OTLP requests reached the endpoint")
			}
			goto done
		}
	}
done:
	if _, ok := seen["/v1/metrics"]; !ok {
		t.Fatalf("metrics were not exported to the endpoint, saw paths %v", seen)
	}
	if _, ok := seen["/v1/logs"]; !ok {
		t.Fatalf("logs were not exported to the endpoint, saw paths %v", seen)
	}
}

func TestDisabledMetricsNeverPanic(t *testing.T) {
	// Just exercise every noop method to prove there are no panics.
	setup, err := otel.Setup(context.Background(), otel.Config{})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	m := setup.Metrics()
	m.CommandHandled("skip", "error")
	m.TrackResolved("youtube", "ok")
	m.TrackResolutionDuration("youtube", 1.5)
	m.PlaybackFailure("stream")
	m.QueueDepth("guild-1", 3)
}
