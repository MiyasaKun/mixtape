package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"fiber_test/internal/application/ports"
)

// PlaybackQueue is the queue boundary used by playback. Keeping this
// interface at the application boundary lets playback work with the real
// Redis-backed queue or an in-memory test double.
type PlaybackQueue interface {
	GetQueue(ctx context.Context, guildID string) (*Queue, error)
	AddToQueue(ctx context.Context, guildID string, track *Track) error
	SaveQueue(ctx context.Context, queue *Queue) error
}

// TrackResolver resolves user input into an application track.
type TrackResolver interface {
	Resolve(ctx context.Context, query string) (*Track, error)
}

// MusicPlaybackService orchestrates audio playback for every guild the bot
// serves. It resolves requested tracks, keeps them in the queue, streams
// them to the guild's voice connection and automatically advances to the
// next track when the current one finishes.
type MusicPlaybackService struct {
	queue         PlaybackQueue
	trackResolver TrackResolver
	streamer      ports.AudioStreamer
	idleTimeout   time.Duration

	mu      sync.Mutex
	players map[string]*player
}

// player holds the playback state for a single guild.
type player struct {
	// advanceMu serializes the "advance to next track" decision so that
	// concurrent triggers (track end, /skip) cannot both advance.
	advanceMu sync.Mutex

	conn     ports.AudioConn
	current  *Track
	provider ports.FrameProvider
	playing  bool
	// cancel aborts the background wait for the currently playing provider.
	cancel context.CancelFunc
	// gen is bumped every time a new track starts or a skip happens. A
	// waiting goroutine captures the generation of its own track and bails
	// out if the generation has moved on (stale auto-advance).
	gen int
	// idleTimer disconnects a connected guild that has no active playback.
	idleTimer *time.Timer
	// idleToken invalidates callbacks from timers that have been reset.
	idleToken uint64
}

const defaultIdleTimeout = 3 * time.Minute

// NewMusicPlaybackService builds the playback service from its dependencies.
func NewMusicPlaybackService(queue PlaybackQueue, trackResolver TrackResolver, streamer ports.AudioStreamer) *MusicPlaybackService {
	return &MusicPlaybackService{
		queue:         queue,
		trackResolver: trackResolver,
		streamer:      streamer,
		idleTimeout:   defaultIdleTimeout,
		players:       make(map[string]*player),
	}
}

// SetVoiceConn registers the voice connection of a guild so playback can be
// routed to it. The Discord command layer calls this after the bot joined a
// channel.
func (s *MusicPlaybackService) SetVoiceConn(guildID string, conn ports.AudioConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.player(guildID)
	p.conn = conn
	s.resetIdleTimerLocked(guildID, p)
}

// Touch resets the inactivity timer for a connected guild.
func (s *MusicPlaybackService) Touch(guildID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetIdleTimerLocked(guildID, s.player(guildID))
}

// Disconnect stops playback and closes the guild's voice connection. It is
// safe to call from command handlers, event handlers, and timer callbacks.
func (s *MusicPlaybackService) Disconnect(ctx context.Context, guildID string) {
	s.mu.Lock()
	p := s.player(guildID)
	s.mu.Unlock()

	p.advanceMu.Lock()
	s.mu.Lock()
	provider, conn := s.clearPlayerLocked(p)
	s.mu.Unlock()
	p.advanceMu.Unlock()

	if provider != nil {
		provider.Close()
	}
	if conn != nil {
		conn.Close(ctx)
	}
}

// Play resolves the query, adds it to the guild's queue and starts playback
// if nothing is playing yet. The returned PlaybackResult describes whether a
// new track started or was merely queued.
func (s *MusicPlaybackService) Play(ctx context.Context, guildID string, query string) (*PlaybackResult, error) {
	s.mu.Lock()
	p := s.player(guildID)
	conn := p.conn
	playing := p.playing
	s.resetIdleTimerLocked(guildID, p)
	s.mu.Unlock()

	if conn == nil {
		return nil, errors.New("bot is not connected to a voice channel")
	}

	track, err := s.trackResolver.Resolve(ctx, query)
	if err != nil {
		return nil, err
	}

	if err := s.queue.AddToQueue(ctx, guildID, track); err != nil {
		return nil, err
	}

	result := &PlaybackResult{Track: track, Started: !playing}

	// Report the 1-based position of the track that was just added.
	queue, err := s.queue.GetQueue(ctx, guildID)
	if err != nil {
		return nil, err
	}
	if queue != nil {
		result.Position = len(queue.Tracks)
	}

	// Only kick off playback when the guild is idle.
	if !playing {
		if err := s.advance(ctx, guildID, 0); err != nil {
			s.stop(guildID, p)
			return nil, err
		}
	}
	return result, nil
}

// Skip stops the current track and advances to the next one in the queue.
func (s *MusicPlaybackService) Skip(ctx context.Context, guildID string) error {
	s.mu.Lock()
	p := s.player(guildID)
	if !p.playing {
		s.mu.Unlock()
		return errors.New("nothing is playing")
	}
	// Bump the generation so any in-flight auto-advance goroutine for the
	// current track considers itself stale and backs off.
	p.gen++
	if p.cancel != nil {
		p.cancel()
	}
	if p.provider != nil {
		p.provider.Close()
	}
	p.playing = false
	s.mu.Unlock()

	err := s.advance(ctx, guildID, 0)
	if err != nil {
		s.stop(guildID, p)
	}
	return err
}

// Stop stops playback entirely. Not yet implemented.
func (s *MusicPlaybackService) Stop(ctx context.Context, guildID string) error {
	return nil
}

// Seek moves playback to the given position. Not yet implemented.
func (s *MusicPlaybackService) Seek(ctx context.Context, guildID string, position int64) error {
	return nil
}

// GetCurrentTrack returns the track currently playing for the guild, or nil
// if nothing is playing.
func (s *MusicPlaybackService) GetCurrentTrack(ctx context.Context, guildID string) (*Track, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.player(guildID)
	if !p.playing {
		return nil, nil
	}
	return p.current, nil
}

// playNext is a convenience wrapper for advance triggered from outside the
// playback loop.
func (s *MusicPlaybackService) playNext(ctx context.Context, guildID string) error {
	return s.advance(ctx, guildID, 0)
}

// advance moves the guild to the next track in the queue and starts streaming
// it. expectedGen is the generation captured by an auto-advance goroutine; a
// nonzero value makes advance a no-op if the generation has moved on, which
// prevents double-advancing after a skip.
func (s *MusicPlaybackService) advance(ctx context.Context, guildID string, expectedGen int) error {
	s.mu.Lock()
	p := s.player(guildID)
	s.mu.Unlock()

	// Serialize the advance decision per guild.
	p.advanceMu.Lock()
	defer p.advanceMu.Unlock()

	// Drop stale auto-advances (e.g. a track was skipped while this advance
	// was already queued).
	if expectedGen != 0 {
		s.mu.Lock()
		stale := p.gen != expectedGen
		s.mu.Unlock()
		if stale {
			return nil
		}
	}

	queue, err := s.queue.GetQueue(ctx, guildID)
	if err != nil {
		return err
	}
	if queue == nil || len(queue.Tracks) == 0 {
		s.stop(guildID, p)
		return nil
	}

	// The next track is the front of the queue. It is streamed first and
	// only removed from the queue once streaming succeeds, so a failed
	// stream keeps the track listed instead of losing it.
	track := queue.Tracks[0]
	provider, err := s.streamer.Stream(ctx, track.URL)
	if err != nil {
		return err
	}

	queue.Tracks = queue.Tracks[1:]
	if err := s.queue.SaveQueue(ctx, queue); err != nil {
		provider.Close()
		return err
	}

	s.mu.Lock()
	p.gen++
	gen := p.gen
	if p.provider != nil {
		p.provider.Close()
	}
	if p.cancel != nil {
		p.cancel()
	}
	advanceCtx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.provider = provider
	p.current = track
	p.playing = true
	// The idle timer may have been armed while the track was resolving. Stop
	// it as soon as playback is active so it can never race with a song.
	s.resetIdleTimerLocked(guildID, p)
	conn := p.conn
	s.mu.Unlock()

	if conn == nil {
		provider.Close()
		return errors.New("bot is not connected to a voice channel")
	}

	// Hand the provider to the voice connection; disgo starts pulling frames.
	conn.SetFrameProvider(provider)

	// Wait for the track to finish (or for it to be skipped/stopped) and then
	// advance to the next track. Uses a background context because it runs
	// long after the originating command's context is gone.
	go func() {
		select {
		case <-provider.Done():
		case <-advanceCtx.Done():
			return
		}
		if err := s.advance(context.Background(), guildID, gen); err != nil {
			s.stop(guildID, p)
		}
	}()

	return nil
}

// stop marks the guild as idle and starts the disconnect timer. Safe to call
// from any goroutine.
func (s *MusicPlaybackService) stop(guildID string, p *player) {
	s.mu.Lock()
	p.playing = false
	p.current = nil
	provider := p.provider
	p.provider = nil
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	s.resetIdleTimerLocked(guildID, p)
	s.mu.Unlock()

	if provider != nil {
		provider.Close()
	}
}

func (s *MusicPlaybackService) resetIdleTimerLocked(guildID string, p *player) {
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}
	p.idleToken++

	if p.conn == nil || p.playing {
		return
	}

	token := p.idleToken
	p.idleTimer = time.AfterFunc(s.idleTimeout, func() {
		s.disconnectIfIdle(guildID, p, token)
	})
}

func (s *MusicPlaybackService) disconnectIfIdle(guildID string, p *player, token uint64) {
	p.advanceMu.Lock()
	s.mu.Lock()
	if p.idleToken != token || p.idleTimer == nil || p.playing || p.conn == nil {
		s.mu.Unlock()
		p.advanceMu.Unlock()
		return
	}

	provider, conn := s.clearPlayerLocked(p)
	s.mu.Unlock()
	p.advanceMu.Unlock()

	if provider != nil {
		provider.Close()
	}
	if conn != nil {
		conn.Close(context.Background())
	}
}

func (s *MusicPlaybackService) clearPlayerLocked(p *player) (ports.FrameProvider, ports.AudioConn) {
	p.gen++
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}
	p.idleToken++
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}

	provider := p.provider
	conn := p.conn
	p.provider = nil
	p.conn = nil
	p.current = nil
	p.playing = false
	return provider, conn
}

// player returns the playback state for a guild, creating it on first use.
// Callers must hold s.mu.
func (s *MusicPlaybackService) player(guildID string) *player {
	p, ok := s.players[guildID]
	if !ok {
		p = &player{}
		s.players[guildID] = p
	}
	return p
}

// PlaybackResult describes the outcome of a Play call.
type PlaybackResult struct {
	Track *Track
	// Position is the 1-based queue position of the added track.
	Position int
	// Started is true if playback began for this track, false if it was only
	// queued behind an already-playing track.
	Started bool
}

// Track is the application-level representation of a playable song.
type Track struct {
	ID           string
	Title        string
	Artist       string
	Duration     int64
	Source       string
	URL          string
	ThumbnailURL string
}
