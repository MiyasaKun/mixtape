//go:build integration

package services_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	redisadapter "fiber_test/internal/adapters/outbound/redis"
	"fiber_test/internal/application/ports"
	"fiber_test/internal/application/services"
)

// telemetryStub records telemetry events from the queue service.
type telemetryStub struct {
	mu      sync.Mutex
	depths  map[string]int
	dropped bool
}

func (s *telemetryStub) CommandHandled(string, string) {}

func (s *telemetryStub) TrackResolved(string, string) {}

func (s *telemetryStub) TrackResolutionDuration(string, float64) {}

func (s *telemetryStub) PlaybackFailure(string) {}

func (s *telemetryStub) QueueDepth(guildID string, depth int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.depths[guildID] = depth
}

var _ ports.Metrics = (*telemetryStub)(nil)

// TestQueueServiceReportsDepth verifies the queue service reports the queue
// depth metric after each modification.
func TestQueueServiceReportsDepth(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := redisadapter.New(redisURL)
	if err != nil {
		t.Skipf("Redis/Valkey is unavailable at %s: %v", redisURL, err)
	}
	defer cache.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	guildID := fmt.Sprintf("telemetry-%d", time.Now().UnixNano())

	stub := &telemetryStub{depths: make(map[string]int)}
	queue := services.NewQueueService(cache, services.Telemetry{Metrics: stub})
	t.Cleanup(func() { _ = queue.ClearQueue(context.Background(), guildID) })

	if err := queue.AddToQueue(ctx, guildID, &services.Track{ID: "t1", Title: "first"}); err != nil {
		t.Fatalf("AddToQueue(first) error = %v", err)
	}
	if err := queue.AddToQueue(ctx, guildID, &services.Track{ID: "t2", Title: "second"}); err != nil {
		t.Fatalf("AddToQueue(second) error = %v", err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.depths[guildID] != 2 {
		t.Fatalf("QueueDepth last value = %d, want 2", stub.depths[guildID])
	}
}
