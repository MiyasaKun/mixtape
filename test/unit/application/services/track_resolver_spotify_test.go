package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/adapters/outbound/youtube"
	"fiber_test/internal/application/services"
)

func TestIsSpotifyURL(t *testing.T) {
	cases := map[string]bool{
		"https://open.spotify.com/track/abc":              true,
		"https://open.spotify.com/album/abc":              true,
		" HTTPS://OPEN.SPOTIFY.COM/track/x ":              true,
		"https://open.spotify.com/intl-de/track/abc?si=x": true,
		// Discord sometimes strips the scheme from pasted links.
		"open.spotify.com/track/abc?si=x":       true,
		"open.spotify.com/intl-de/track/abc":    true,
		"www.open.spotify.com/track/abc":        true,
		"https://youtube.com/watch?v=x":         false,
		"https://soundcloud.com/a/b":            false,
		"just a search query":                   false,
		"https://open.spotify.com.evil/track":   false,
		"https://notopen.spotify.com/track/abc": false,
	}
	for input, want := range cases {
		if got := services.IsSpotifyURL(input); got != want {
			t.Errorf("IsSpotifyURL(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestExtractSpotifyTrackID(t *testing.T) {
	valid := []string{
		"https://open.spotify.com/track/tr-1",
		"https://open.spotify.com/track/tr-1?si=abc123",
		"https://open.spotify.com/track/tr-1#anchor",
		"https://open.spotify.com/intl-de/track/tr-1?si=65c1b1557e5b481f",
		"https://open.spotify.com/intl-pt/track/tr-1",
		// Scheme-less links (Discord strips the scheme from pasted text).
		"open.spotify.com/track/tr-1?si=abc",
		"www.open.spotify.com/intl-de/track/tr-1",
	}
	for _, input := range valid {
		if got := services.ExtractSpotifyTrackID(input); got != "tr-1" {
			t.Errorf("ExtractSpotifyTrackID(%q) = %q, want tr-1", input, got)
		}
	}

	// Spotify IDs are base62 and case-sensitive; the case must survive.
	mixed := "https://open.spotify.com/intl-de/track/4naUDV97XDe3gnHbKLlwPZ?si=65c1b1557e5b481f"
	if got := services.ExtractSpotifyTrackID(mixed); got != "4naUDV97XDe3gnHbKLlwPZ" {
		t.Errorf("ExtractSpotifyTrackID(%q) = %q, want case preserved", mixed, got)
	}
	invalid := []string{
		"https://open.spotify.com/album/abc",
		"https://open.spotify.com/intl-de/album/abc",
		"https://open.spotify.com/track/",
		"just a search query",
	}
	for _, input := range invalid {
		if got := services.ExtractSpotifyTrackID(input); got != "" {
			t.Errorf("ExtractSpotifyTrackID(%q) = %q, want empty", input, got)
		}
	}
}

func TestSpotifyYouTubeOptionsSearchesYouTubeForTrack(t *testing.T) {
	ytRunner := func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) != 4 || !strings.Contains(args[3], "ytsearch5:Kavinsky - Nightcall") {
			t.Errorf("unexpected yt-dlp args: %v", args)
		}
		return json.Marshal(map[string]any{
			"entries": []any{
				map[string]any{"id": "vid-1", "title": "Nightcall (Official Audio)", "channel": "Kavinsky", "duration": 258, "webpage_url": "https://www.youtube.com/watch?v=vid-1"},
				map[string]any{"id": "vid-2", "title": "Nightcall - Live", "channel": "Kavinsky TV", "duration": 292, "webpage_url": "https://www.youtube.com/watch?v=vid-2"},
			},
		})
	}

	resolver := newSpotifyResolver(t, ytRunner)

	tracks, coverURL, err := resolver.SpotifyYouTubeOptions(context.Background(), "https://open.spotify.com/track/tr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if coverURL != "https://i.scdn.co/outrun.jpg" {
		t.Errorf("coverURL = %q, want spotify album image", coverURL)
	}

	if len(tracks) != 2 {
		t.Fatalf("want 2 options, got %d", len(tracks))
	}
	for _, track := range tracks {
		if track.Source != "youtube" {
			t.Errorf("want youtube source, got %q", track.Source)
		}
		if track.URL == "" {
			t.Error("want track URL set for playback")
		}
	}
	if tracks[0].Title != "Nightcall (Official Audio)" {
		t.Errorf("unexpected first option: %+v", tracks[0])
	}
}

func TestSpotifyYouTubeOptionsRejectsNonTrackURLs(t *testing.T) {
	resolver := newSpotifyResolver(t, nil)

	_, _, err := resolver.SpotifyYouTubeOptions(context.Background(), "https://open.spotify.com/album/abc")
	if err == nil {
		t.Fatal("want error for album URL")
	}
	if !strings.Contains(err.Error(), "track") {
		t.Errorf("error = %q, want mention of track URLs", err)
	}
}

func newSpotifyResolver(t *testing.T, ytRunner func(ctx context.Context, args ...string) ([]byte, error)) *services.TrackResolverService {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/token":
			writeJSON(w, map[string]any{"access_token": "app-token", "expires_in": 3600})
		case "/v1/tracks/tr-1":
			writeJSON(w, map[string]any{
				"id": "tr-1", "name": "Nightcall", "duration_ms": 258000,
				"artists": []any{map[string]any{"name": "Kavinsky"}},
				"album":   map[string]any{"name": "OutRun", "images": []any{map[string]any{"url": "https://i.scdn.co/outrun.jpg"}}},
				"external_urls": map[string]any{"spotify": "https://open.spotify.com/track/tr-1"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	sp := spotify.New("client-id", "client-secret", "http://localhost/cb", spotify.WithEndpoints(srv.URL, srv.URL+"/v1"))
	yt := youtube.New(youtube.WithRunner(ytRunner))
	return services.NewTrackResolverService(yt, sp, nil)
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
