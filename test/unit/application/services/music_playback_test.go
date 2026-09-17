package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/application/services"
)

type fakeQueue struct {
	mu       sync.Mutex
	queue    services.Queue
	saveCall int
}

func (q *fakeQueue) GetQueue(context.Context, string) (*services.Queue, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.queue.GuildID == "" {
		return nil, nil
	}
	return &q.queue, nil
}

func (q *fakeQueue) AddToQueue(_ context.Context, guildID string, track *services.Track) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.queue.GuildID == "" {
		q.queue.GuildID = guildID
	}
	q.queue.Tracks = append(q.queue.Tracks, track)
	return nil
}

func (q *fakeQueue) SaveQueue(_ context.Context, queue *services.Queue) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.queue = *queue
	q.saveCall++
	return nil
}

type fakeResolver struct {
	track *services.Track
}

func (r *fakeResolver) Resolve(context.Context, string) (*services.Track, error) {
	return r.track, nil
}

type fakeStreamer struct {
	provider *fakeProvider
}

func (s *fakeStreamer) Stream(context.Context, string) (ports.FrameProvider, error) {
	return s.provider, nil
}

type fakeProvider struct {
	done chan struct{}
	once sync.Once
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{done: make(chan struct{})}
}

func (p *fakeProvider) ProvideFrame() ([]byte, error) {
	return []byte{1}, nil
}

func (p *fakeProvider) Done() <-chan struct{} {
	return p.done
}

func (p *fakeProvider) Close() {
	p.once.Do(func() { close(p.done) })
}

type fakeConn struct {
	mu        sync.Mutex
	providers []ports.FrameProvider
	closed    bool
}

func (c *fakeConn) SetFrameProvider(provider ports.FrameProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers = append(c.providers, provider)
}

func (c *fakeConn) Close(context.Context) {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
}

func TestPlaybackStartsAndAdvancesWhenProviderEnds(t *testing.T) {
	provider := newFakeProvider()
	queue := &fakeQueue{}
	conn := &fakeConn{}
	track := &services.Track{ID: "track-1", Title: "Test track", URL: "https://example.test/track"}
	playback := services.NewMusicPlaybackService(
		queue,
		&fakeResolver{track: track},
		&fakeStreamer{provider: provider},
	)
	t.Cleanup(func() { playback.Disconnect(context.Background(), "guild-1") })
	playback.SetVoiceConn("guild-1", conn)

	result, err := playback.Play(context.Background(), "guild-1", "test query")
	if err != nil {
		t.Fatalf("Play() error = %v", err)
	}
	if !result.Started {
		t.Fatal("Play() did not report playback as started")
	}

	if current, err := playback.GetCurrentTrack(context.Background(), "guild-1"); err != nil || current != track {
		t.Fatalf("current track = %v, error = %v", current, err)
	}
	if queue.saveCall != 1 || len(queue.queue.Tracks) != 0 {
		t.Fatalf("queue was not consumed on start: saves=%d tracks=%d", queue.saveCall, len(queue.queue.Tracks))
	}

	provider.Close()
	waitFor(t, func() bool {
		current, _ := playback.GetCurrentTrack(context.Background(), "guild-1")
		return current == nil
	})
}

func TestDisconnectStopsProviderAndConnection(t *testing.T) {
	provider := newFakeProvider()
	conn := &fakeConn{}
	playback := services.NewMusicPlaybackService(
		&fakeQueue{},
		&fakeResolver{track: &services.Track{Title: "Test track"}},
		&fakeStreamer{provider: provider},
	)
	playback.SetVoiceConn("guild-1", conn)

	if _, err := playback.Play(context.Background(), "guild-1", "test query"); err != nil {
		t.Fatalf("Play() error = %v", err)
	}

	playback.Disconnect(context.Background(), "guild-1")

	select {
	case <-provider.done:
	case <-time.After(time.Second):
		t.Fatal("Disconnect() did not stop the provider")
	}

	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()
	if !closed {
		t.Fatal("Disconnect() did not close the connection")
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
