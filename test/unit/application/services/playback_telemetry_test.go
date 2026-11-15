package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/application/services"
)

// stubMetrics records telemetry events so tests can assert what services emit.
type stubMetrics struct {
	mu               sync.Mutex
	playbackFailures map[string]int
	tracksResolved   map[string]int
	lastDuration     float64
	queueDepthGuild  string
	queueDepth       int
	commandsHandled  map[string]int
}

func newStubMetrics() *stubMetrics {
	return &stubMetrics{
		playbackFailures: make(map[string]int),
		tracksResolved:   make(map[string]int),
		commandsHandled:  make(map[string]int),
	}
}

func (m *stubMetrics) CommandHandled(command, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandsHandled[command+"|"+status]++
}

func (m *stubMetrics) TrackResolved(provider, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tracksResolved[provider+"|"+status]++
}

func (m *stubMetrics) TrackResolutionDuration(provider string, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastDuration = seconds
}

func (m *stubMetrics) PlaybackFailure(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playbackFailures[reason]++
}

func (m *stubMetrics) QueueDepth(guildID string, depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueDepthGuild = guildID
	m.queueDepth = depth
}

type failingResolver struct{}

func (failingResolver) Resolve(context.Context, string) (*services.Track, error) {
	return nil, errors.New("no matching video found")
}

func TestPlayReportsResolveFailureWithTelemetry(t *testing.T) {
	metrics := newStubMetrics()
	playback := services.NewMusicPlaybackService(
		&fakeQueue{},
		failingResolver{},
		&fakeStreamer{provider: newFakeProvider()},
		services.Telemetry{Metrics: metrics},
	)
	playback.SetVoiceConn("guild-1", &fakeConn{})

	_, err := playback.Play(context.Background(), "guild-1", "some query")
	if err == nil {
		t.Fatal("Play() should fail when resolvable fails")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.playbackFailures["resolve"] != 1 {
		t.Fatalf("PlaybackFailure calls = %v, want 1 for reason %q", metrics.playbackFailures, "resolve")
	}
}

func TestPlayReportsQueueDepthAfterAdd(t *testing.T) {
	metrics := newStubMetrics()
	track := &services.Track{ID: "t1", URL: "https://example.test/track"}
	playback := services.NewMusicPlaybackService(
		&fakeQueue{},
		&fakeResolver{track: track},
		&fakeStreamer{provider: newFakeProvider()},
		services.Telemetry{Metrics: metrics},
	)
	playback.SetVoiceConn("guild-1", &fakeConn{})

	if _, err := playback.Play(context.Background(), "guild-1", "test"); err != nil {
		t.Fatalf("Play() error = %v", err)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.queueDepthGuild != "guild-1" || metrics.queueDepth != 1 {
		t.Fatalf("QueueDepth = (%q, %d), want (guild-1, 1)", metrics.queueDepthGuild, metrics.queueDepth)
	}
}

var _ ports.Metrics = (*stubMetrics)(nil)
