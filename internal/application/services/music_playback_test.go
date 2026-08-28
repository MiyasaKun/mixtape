package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"fiber_test/internal/application/ports"
)

type testAudioConn struct {
	closed chan struct{}
	once   sync.Once
}

func newTestAudioConn() *testAudioConn {
	return &testAudioConn{closed: make(chan struct{})}
}

func (c *testAudioConn) SetFrameProvider(ports.FrameProvider) {}

func (c *testAudioConn) Close(context.Context) {
	c.once.Do(func() { close(c.closed) })
}

type testFrameProvider struct {
	done chan struct{}
	once sync.Once
}

func newTestFrameProvider() *testFrameProvider {
	return &testFrameProvider{done: make(chan struct{})}
}

func (p *testFrameProvider) ProvideFrame() ([]byte, error) { return nil, nil }

func (p *testFrameProvider) Done() <-chan struct{} { return p.done }

func (p *testFrameProvider) Close() {
	p.once.Do(func() { close(p.done) })
}

func newTestPlaybackService(timeout time.Duration) *MusicPlaybackService {
	return &MusicPlaybackService{
		idleTimeout: timeout,
		players:     make(map[string]*player),
	}
}

func TestIdleConnectionDisconnectsAfterTimeout(t *testing.T) {
	service := newTestPlaybackService(10 * time.Millisecond)
	conn := newTestAudioConn()
	service.SetVoiceConn("guild", conn)

	select {
	case <-conn.closed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("idle connection was not disconnected")
	}

	service.mu.Lock()
	p := service.player("guild")
	connected := p.conn != nil
	playing := p.playing
	service.mu.Unlock()

	if connected {
		t.Fatal("idle connection remained registered")
	}
	if playing {
		t.Fatal("idle player remained active")
	}
}

func TestStoppedPlaybackDisconnectsAfterTimeout(t *testing.T) {
	service := newTestPlaybackService(10 * time.Millisecond)
	conn := newTestAudioConn()
	service.SetVoiceConn("guild", conn)

	service.mu.Lock()
	p := service.player("guild")
	p.playing = true
	service.mu.Unlock()
	service.stop("guild", p)

	select {
	case <-conn.closed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("stopped playback did not start the idle disconnect timer")
	}
}

func TestStaleIdleTimerDoesNotDisconnectConnection(t *testing.T) {
	service := newTestPlaybackService(time.Second)
	conn := newTestAudioConn()
	service.SetVoiceConn("guild", conn)

	service.mu.Lock()
	p := service.player("guild")
	staleToken := p.idleToken
	service.mu.Unlock()

	service.SetVoiceConn("guild", conn)
	service.disconnectIfIdle("guild", p, staleToken)

	select {
	case <-conn.closed:
		t.Fatal("stale idle timer disconnected the connection")
	default:
	}

	service.Disconnect(context.Background(), "guild")
}

func TestActivityResetsIdleTimer(t *testing.T) {
	service := newTestPlaybackService(time.Second)
	conn := newTestAudioConn()
	service.SetVoiceConn("guild", conn)

	service.mu.Lock()
	p := service.player("guild")
	staleToken := p.idleToken
	service.mu.Unlock()

	service.Touch("guild")
	service.disconnectIfIdle("guild", p, staleToken)

	select {
	case <-conn.closed:
		t.Fatal("activity did not invalidate the idle timer")
	default:
	}

	service.Disconnect(context.Background(), "guild")
}

func TestDisconnectCleansPlaybackState(t *testing.T) {
	service := newTestPlaybackService(time.Second)
	conn := newTestAudioConn()
	provider := newTestFrameProvider()
	service.SetVoiceConn("guild", conn)

	service.mu.Lock()
	p := service.player("guild")
	p.provider = provider
	p.current = &Track{Title: "track"}
	p.playing = true
	service.mu.Unlock()

	service.Disconnect(context.Background(), "guild")

	select {
	case <-conn.closed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("connection was not closed")
	}
	select {
	case <-provider.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("provider was not closed")
	}

	service.mu.Lock()
	connected := p.conn != nil
	current := p.current != nil
	playing := p.playing
	service.mu.Unlock()
	if connected || current || playing {
		t.Fatal("disconnect left stale playback state")
	}
}
