package services_test

import (
	"context"
	"testing"
	"time"

	"fiber_test/internal/application/services"
)

// frameDelay is the fixed cadence of the voice sender: one frame per 20ms.
const frameDelay = 20 * time.Millisecond

func TestGetPlaybackStatusCountsFrames(t *testing.T) {
	provider := newFakeProvider()
	conn := &fakeConn{}
	track := &services.Track{ID: "track-1", Title: "Test track", URL: "https://example.test/track", Duration: 180}
	playback := services.NewMusicPlaybackService(
		&fakeQueue{},
		&fakeResolver{track: track},
		&fakeStreamer{provider: provider},
	)
	t.Cleanup(func() { playback.Disconnect(context.Background(), "guild-1") })
	playback.SetVoiceConn("guild-1", conn)

	if _, err := playback.Play(context.Background(), "guild-1", "test query"); err != nil {
		t.Fatalf("Play() error = %v", err)
	}

	// The connection received the wrapped provider; pull 25 frames through
	// it, which must count as 25 × 20ms of elapsed playback.
	wrapped := conn.lastProvider()
	if wrapped == nil {
		t.Fatal("no frame provider handed to the connection")
	}
	const frames = 25
	for i := 0; i < frames; i++ {
		if _, err := wrapped.ProvideFrame(); err != nil {
			t.Fatalf("ProvideFrame() error = %v", err)
		}
	}

	status, err := playback.GetPlaybackStatus(context.Background(), "guild-1")
	if err != nil {
		t.Fatalf("GetPlaybackStatus() error = %v", err)
	}
	if status == nil {
		t.Fatal("GetPlaybackStatus() = nil while playing")
	}
	if status.Track != track {
		t.Errorf("status.Track = %v, want the played track", status.Track)
	}
	if want := frames * frameDelay; status.Elapsed != want {
		t.Errorf("status.Elapsed = %v, want %v", status.Elapsed, want)
	}
}

func TestGetPlaybackStatusNilWhenIdle(t *testing.T) {
	playback := services.NewMusicPlaybackService(&fakeQueue{}, &fakeResolver{}, &fakeStreamer{})

	status, err := playback.GetPlaybackStatus(context.Background(), "guild-1")
	if err != nil {
		t.Fatalf("GetPlaybackStatus() error = %v", err)
	}
	if status != nil {
		t.Fatalf("GetPlaybackStatus() = %+v, want nil when idle", status)
	}
}

func TestPlayWithCoverOverridesThumbnailAndSource(t *testing.T) {
	provider := newFakeProvider()
	conn := &fakeConn{}
	resolved := &services.Track{ID: "track-1", Title: "Nightcall", URL: "https://example.test/track", ThumbnailURL: "yt-thumb.jpg", Source: "youtube"}
	queue := &fakeQueue{}
	playback := services.NewMusicPlaybackService(
		queue,
		&fakeResolver{track: resolved},
		&fakeStreamer{provider: provider},
	)
	t.Cleanup(func() { playback.Disconnect(context.Background(), "guild-1") })
	playback.SetVoiceConn("guild-1", conn)

	cover := "https://i.scdn.co/album.jpg"
	if _, err := playback.PlayWithCover(context.Background(), "guild-1", "test query", cover); err != nil {
		t.Fatalf("PlayWithCover() error = %v", err)
	}

	status, err := playback.GetPlaybackStatus(context.Background(), "guild-1")
	if err != nil || status == nil {
		t.Fatalf("GetPlaybackStatus() = %v, %v", status, err)
	}
	played := status.Track
	if played.ThumbnailURL != cover {
		t.Errorf("ThumbnailURL = %q, want the spotify cover %q", played.ThumbnailURL, cover)
	}
	if played.Source != "spotify" {
		t.Errorf("Source = %q, want %q when a spotify cover is applied", played.Source, "spotify")
	}
}

func TestPlayWithCoverKeepsYouTubeDefaultsWhenCoverEmpty(t *testing.T) {
	provider := newFakeProvider()
	conn := &fakeConn{}
	resolved := &services.Track{ID: "track-1", Title: "Song", URL: "https://example.test/track", ThumbnailURL: "yt-thumb.jpg", Source: "youtube"}
	playback := services.NewMusicPlaybackService(
		&fakeQueue{},
		&fakeResolver{track: resolved},
		&fakeStreamer{provider: provider},
	)
	t.Cleanup(func() { playback.Disconnect(context.Background(), "guild-1") })
	playback.SetVoiceConn("guild-1", conn)

	if _, err := playback.PlayWithCover(context.Background(), "guild-1", "test query", ""); err != nil {
		t.Fatalf("PlayWithCover() error = %v", err)
	}
	if resolved.ThumbnailURL != "yt-thumb.jpg" || resolved.Source != "youtube" {
		t.Errorf("empty cover must not override track, got %+v", resolved)
	}
}
