package services_test

import (
	"context"
	"errors"
	"testing"

	"fiber_test/internal/application/services"
)

type fakeCandidateResolver struct {
	candidates []services.Track
	coverURL   string
	err        error
	gotURL     string
}

func (f *fakeCandidateResolver) SpotifyYouTubeOptions(ctx context.Context, spotifyURL string) ([]services.Track, string, error) {
	f.gotURL = spotifyURL
	return f.candidates, f.coverURL, f.err
}

type fakePlaybackStarter struct {
	result   *services.PlaybackResult
	err      error
	gotGuild string
	gotQuery string
	gotCover string
	calls    int
}

func (f *fakePlaybackStarter) PlayWithCover(ctx context.Context, guildID, query, coverURL string) (*services.PlaybackResult, error) {
	f.calls++
	f.gotGuild, f.gotQuery, f.gotCover = guildID, query, coverURL
	return f.result, f.err
}

func TestEnqueueSpotifyPlaysBestYouTubeMatchWithSpotifyCover(t *testing.T) {
	resolver := &fakeCandidateResolver{
		candidates: []services.Track{
			{Title: "First match", URL: "https://youtube.com/watch?v=first"},
			{Title: "Second match", URL: "https://youtube.com/watch?v=second"},
		},
		coverURL: "https://i.scdn.co/image/cover",
	}
	starter := &fakePlaybackStarter{result: &services.PlaybackResult{
		Track:    &services.Track{Title: "First match"},
		Started:  true,
		Position: 1,
	}}
	svc := services.NewWebPlaybackService(resolver, starter)

	result, err := svc.EnqueueSpotify(context.Background(), "guild-1", "https://open.spotify.com/track/abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Track.Title != "First match" || !result.Started {
		t.Errorf("unexpected result: %+v", result)
	}
	if resolver.gotURL != "https://open.spotify.com/track/abc" {
		t.Errorf("want spotify url forwarded, got %q", resolver.gotURL)
	}
	if starter.gotGuild != "guild-1" {
		t.Errorf("want guild forwarded, got %q", starter.gotGuild)
	}
	if starter.gotQuery != "https://youtube.com/watch?v=first" {
		t.Errorf("want best match played, got %q", starter.gotQuery)
	}
	if starter.gotCover != "https://i.scdn.co/image/cover" {
		t.Errorf("want spotify cover forwarded, got %q", starter.gotCover)
	}
}

func TestEnqueueSpotifyFailsWhenNoCandidateMatches(t *testing.T) {
	starter := &fakePlaybackStarter{}
	svc := services.NewWebPlaybackService(&fakeCandidateResolver{}, starter)

	_, err := svc.EnqueueSpotify(context.Background(), "guild-1", "https://open.spotify.com/track/abc")
	if !errors.Is(err, services.ErrNoPlayableMatch) {
		t.Fatalf("want ErrNoPlayableMatch, got %v", err)
	}
	if starter.calls != 0 {
		t.Error("playback must not be started without a candidate")
	}
}

func TestEnqueueSpotifyReturnsResolverErrors(t *testing.T) {
	want := errors.New("spotify lookup failed")
	svc := services.NewWebPlaybackService(&fakeCandidateResolver{err: want}, &fakePlaybackStarter{})

	_, err := svc.EnqueueSpotify(context.Background(), "guild-1", "https://open.spotify.com/track/abc")
	if !errors.Is(err, want) {
		t.Fatalf("want wrapped resolver error, got %v", err)
	}
}
