//go:build integration

package services_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	redisadapter "fiber_test/internal/adapters/outbound/redis"
	"fiber_test/internal/application/services"
)

func TestQueueServicePersistsAcrossInstances(t *testing.T) {
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
	guildID := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	queue := services.NewQueueService(cache)
	t.Cleanup(func() { _ = queue.ClearQueue(context.Background(), guildID) })

	first := &services.Track{ID: "track-1", Title: "First track"}
	second := &services.Track{ID: "track-2", Title: "Second track"}
	if err := queue.AddToQueue(ctx, guildID, first); err != nil {
		t.Fatalf("AddToQueue(first) error = %v", err)
	}
	if err := queue.AddToQueue(ctx, guildID, second); err != nil {
		t.Fatalf("AddToQueue(second) error = %v", err)
	}

	reloaded := services.NewQueueService(cache)
	stored, err := reloaded.GetQueue(ctx, guildID)
	if err != nil {
		t.Fatalf("GetQueue() error = %v", err)
	}
	if stored == nil || len(stored.Tracks) != 2 {
		t.Fatalf("stored queue = %#v", stored)
	}
	if stored.Tracks[0].ID != first.ID || stored.Tracks[1].ID != second.ID {
		t.Fatalf("stored tracks = %#v", stored.Tracks)
	}

	stored.Tracks = stored.Tracks[1:]
	if err := reloaded.SaveQueue(ctx, stored); err != nil {
		t.Fatalf("SaveQueue() error = %v", err)
	}
	remaining, err := queue.GetQueue(ctx, guildID)
	if err != nil {
		t.Fatalf("GetQueue() after SaveQueue error = %v", err)
	}
	if len(remaining.Tracks) != 1 || remaining.Tracks[0].ID != second.ID {
		t.Fatalf("remaining queue = %#v", remaining)
	}
}
