package spotify_test

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGetTrackUsesClientCredentialsAndMapsTrack(t *testing.T) {
	var tokenRequests int32
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/token":
			atomic.AddInt32(&tokenRequests, 1)
			id, secret, ok := r.BasicAuth()
			if !ok || id != "client-id" || secret != "client-secret" {
				t.Errorf("want client basic auth, got id=%q secret=%q ok=%v", id, secret, ok)
			}
			if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
				t.Errorf("want form content type, got %q", ct)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
				t.Errorf("want client_credentials grant, got %q", got)
			}
			writeJSON(t, w, map[string]any{"access_token": "app-token", "expires_in": 3600})
		case r.URL.Path == "/v1/tracks/tr-1":
			if got := r.Header.Get("Authorization"); got != "Bearer app-token" {
				t.Errorf("want bearer app-token, got %q", got)
			}
			writeJSON(t, w, map[string]any{
				"id":          "tr-1",
				"name":        "Nightcall",
				"duration_ms": 258000,
				"artists":     []any{map[string]any{"name": "Kavinsky"}, map[string]any{"name": "Lovefoxxx"}},
				"album": map[string]any{
					"name":   "OutRun",
					"images": []any{map[string]any{"url": "https://img.example/outrun.jpg"}},
				},
				"external_urls": map[string]any{"spotify": "https://open.spotify.com/track/tr-1"},
			})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
		}
	})

	track, err := client.GetTrack(context.Background(), "tr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if track.ID != "tr-1" || track.Title != "Nightcall" || track.Album != "OutRun" {
		t.Errorf("unexpected track: %+v", track)
	}
	if len(track.Artists) != 2 || track.Artists[0] != "Kavinsky" {
		t.Errorf("want mapped artists, got %v", track.Artists)
	}
	if track.DurationMS != 258000 {
		t.Errorf("want duration 258000, got %d", track.DurationMS)
	}
	if track.ImageURL != "https://img.example/outrun.jpg" {
		t.Errorf("want album image, got %q", track.ImageURL)
	}
	if atomic.LoadInt32(&tokenRequests) != 1 {
		t.Errorf("want exactly 1 token request, got %d", tokenRequests)
	}
}

func TestGetTrackCachesClientCredentialsToken(t *testing.T) {
	var tokenRequests int32
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token" {
			atomic.AddInt32(&tokenRequests, 1)
			writeJSON(t, w, map[string]any{"access_token": "app-token", "expires_in": 3600})
			return
		}
		writeJSON(t, w, map[string]any{"id": "tr-1", "name": "Nightcall", "duration_ms": 258000, "artists": []any{}})
	})

	for i := 0; i < 3; i++ {
		if _, err := client.GetTrack(context.Background(), "tr-1"); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if atomic.LoadInt32(&tokenRequests) != 1 {
		t.Errorf("want token fetched once across calls, got %d", tokenRequests)
	}
}
