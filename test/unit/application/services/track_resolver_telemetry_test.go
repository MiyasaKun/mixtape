package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"fiber_test/internal/adapters/outbound/youtube"
	"fiber_test/internal/application/services"
)

type fakeYouTube struct {
	err   error
	track *youtube.Track
}

func (f *fakeYouTube) Resolve(context.Context, string) (*youtube.Track, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.track, nil
}

func TestTrackResolverRecordsSuccessfulResolution(t *testing.T) {
	metrics := newStubMetrics()
	resolver := services.NewTrackResolverService(
		&fakeYouTube{track: &youtube.Track{ID: "yt1", Title: "Song", URL: "https://youtube.com/watch?v=yt1"}},
		nil,
		nil,
		services.Telemetry{Metrics: metrics},
	)

	track, err := resolver.Resolve(context.Background(), "https://youtube.com/watch?v=yt1")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if track.Source != "youtube" {
		t.Fatalf("Resolve() source = %q, want youtube", track.Source)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.tracksResolved["youtube|ok"] != 1 {
		t.Fatalf("TrackResolved statuses = %v, want youtube|ok recorded once", metrics.tracksResolved)
	}
	if metrics.lastDuration <= 0 {
		t.Fatalf("TrackResolutionDuration = %f, want > 0", metrics.lastDuration)
	}
}

func TestTrackResolverRecordsFailedResolution(t *testing.T) {
	metrics := newStubMetrics()
	resolver := services.NewTrackResolverService(
		&fakeYouTube{err: errors.New("yt-dlp exited with code 101")},
		nil,
		nil,
		services.Telemetry{Metrics: metrics},
	)

	_, err := resolver.Resolve(context.Background(), "https://youtube.com/watch?v=bad")
	if err == nil {
		t.Fatal("Resolve() should surface the adapter error")
	}
	if !strings.Contains(err.Error(), "resolve youtube track") {
		t.Fatalf("error should be wrapped with context, got %q", err.Error())
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.tracksResolved["youtube|error"] != 1 {
		t.Fatalf("TrackResolved statuses = %v, want youtube|error recorded once", metrics.tracksResolved)
	}
	if metrics.lastDuration <= 0 {
		t.Fatalf("TrackResolutionDuration should be measured even on failure, got %f", metrics.lastDuration)
	}
}

func TestTrackResolverMarksUnsupportedSources(t *testing.T) {
	metrics := newStubMetrics()
	resolver := services.NewTrackResolverService(nil, nil, nil, services.Telemetry{Metrics: metrics})

	_, err := resolver.Resolve(context.Background(), "https://open.spotify.com/track/example")
	if err == nil {
		t.Fatal("spotify should be unsupported")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.tracksResolved["spotify|unsupported"] != 1 {
		t.Fatalf("TrackResolved statuses = %v, want spotify|unsupported once", metrics.tracksResolved)
	}
}
