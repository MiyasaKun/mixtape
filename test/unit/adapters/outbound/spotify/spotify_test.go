package spotify_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/domain"
)

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func newTestSpotify(t *testing.T, handler http.HandlerFunc) *spotify.Spotify {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return spotify.New("client-id", "client-secret", "http://localhost/cb", spotify.WithEndpoints(srv.URL, srv.URL+"/v1"))
}

func TestListPlaylistsMapsPayloadAndSignalsNextPage(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me/playlists" {
			t.Errorf("want /v1/me/playlists, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Errorf("want bearer access token, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("want limit 50, got %q", got)
		}
		if got := r.URL.Query().Get("offset"); got != "0" {
			t.Errorf("want offset 0, got %q", got)
		}
		writeJSON(t, w, map[string]any{
			"items": []any{
				map[string]any{
					"id":            "pl-1",
					"name":          "Deep Focus",
					"description":   "Long sessions",
					"collaborative": true,
					"owner":         map[string]any{"id": "sp-1", "display_name": "Ada"},
					"items":         map[string]any{"total": 12},
					"images":        []any{map[string]any{"url": "https://img.example/cover.jpg"}},
					"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-1"},
				},
				map[string]any{
					// Legacy field name: still parsed for older responses.
					"id":            "pl-2",
					"name":          "Chill",
					"collaborative": false,
					"owner":         map[string]any{"id": "someone-else", "display_name": "Grace"},
					"tracks":        map[string]any{"total": 3},
					"images":        []any{},
					"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-2"},
				},
			},
			"next": "https://api.spotify.com/v1/me/playlists?limit=50&offset=50",
		})
	})

	page, err := client.ListPlaylists(context.Background(), "access-token", "sp-1", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2 playlists, got %d", len(page.Items))
	}

	first := page.Items[0]
	if first.ID != "pl-1" || first.Name != "Deep Focus" {
		t.Errorf("unexpected first playlist: %+v", first)
	}
	if first.Owner != "Ada" || first.TrackCount != 12 || !first.Collaborative {
		t.Errorf("unexpected first playlist metadata: %+v", first)
	}
	if first.ImageURL != "https://img.example/cover.jpg" {
		t.Errorf("want cover image, got %q", first.ImageURL)
	}
	if !first.IsOwned {
		t.Error("want owned flag for playlist owned by the connected user")
	}
	if page.Items[1].IsOwned {
		t.Error("want not-owned for a playlist owned by someone else")
	}
	if page.Items[1].TrackCount != 3 {
		t.Errorf("want legacy tracks.total parsed as 3, got %d", page.Items[1].TrackCount)
	}
	if !page.Items[0].Browseable {
		t.Error("want browseable when Spotify returns the items field")
	}
	if page.Items[1].Browseable {
		t.Error("want not browseable when the items field is absent")
	}

	if page.Next == nil || *page.Next != 50 {
		t.Errorf("want next offset 50, got %v", page.Next)
	}
}

func TestListPlaylistsHasNoNextPageWhenExhausted(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"items": []any{}, "next": nil})
	})

	page, err := client.ListPlaylists(context.Background(), "access-token", "sp-1", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Next != nil {
		t.Errorf("want no next page, got %v", *page.Next)
	}
}

func TestListLikedTracksMapsSavedTracks(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me/tracks" {
			t.Errorf("want /v1/me/tracks, got %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"items": []any{
				map[string]any{
					"added_at": "2026-01-01T00:00:00Z",
					"track": map[string]any{
						"id":          "tr-1",
						"name":        "Nightcall",
						"duration_ms": 258000,
						"artists":     []any{map[string]any{"name": "Kavinsky"}, map[string]any{"name": "Lovefoxxx"}},
						"album": map[string]any{
							"name":   "OutRun",
							"images": []any{map[string]any{"url": "https://img.example/outrun.jpg"}},
						},
						"external_urls": map[string]any{"spotify": "https://open.spotify.com/track/tr-1"},
					},
				},
			},
			"next": "",
		})
	})

	page, err := client.ListLikedTracks(context.Background(), "access-token", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("want 1 track, got %d", len(page.Items))
	}
	track := page.Items[0]
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
	if page.Next != nil {
		t.Error("want no next page when spotify returns no next url")
	}
}

func TestListPlaylistTracksTargetsPlaylist(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/playlists/pl-9/items" {
			t.Errorf("want playlist items path, got %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"items": []any{
				map[string]any{
					"item": map[string]any{
						"id":            "tr-9",
						"name":          "Resonance",
						"duration_ms":   200000,
						"artists":       []any{map[string]any{"name": "HOME"}},
						"album":         map[string]any{"name": "Odyssey", "images": []any{}},
						"external_urls": map[string]any{"spotify": "https://open.spotify.com/track/tr-9"},
					},
				},
			},
		})
	})

	page, err := client.ListPlaylistTracks(context.Background(), "access-token", "pl-9", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Resonance" {
		t.Fatalf("unexpected tracks: %+v", page.Items)
	}
}

func TestRequestErrorsCarryUpstreamStatus(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(t, w, map[string]any{"error": map[string]any{"status": 403, "message": "Forbidden"}})
	})

	_, err := client.ListPlaylistTracks(context.Background(), "access-token", "pl-1", 0, 100)

	var apiErr *domain.SpotifyAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *domain.SpotifyAPIError, got %v", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("want status 403, got %d", apiErr.StatusCode)
	}
}

func TestRefreshAccessTokenSendsRefreshGrant(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			t.Errorf("want /api/token, got %s", r.URL.Path)
		}
		id, secret, ok := r.BasicAuth()
		if !ok || id != "client-id" || secret != "client-secret" {
			t.Errorf("want client basic auth, got id=%q secret=%q ok=%v", id, secret, ok)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "grant_type=refresh_token") {
			t.Errorf("want refresh_token grant, got %s", body)
		}
		if !strings.Contains(string(body), "refresh_token=stored-refresh") {
			t.Errorf("want stored refresh token, got %s", body)
		}
		writeJSON(t, w, map[string]any{"access_token": "fresh-access", "expires_in": 3600})
	})

	before := time.Now()
	tokens, err := client.RefreshAccessToken(context.Background(), "stored-refresh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.AccessToken != "fresh-access" {
		t.Errorf("want fresh-access, got %q", tokens.AccessToken)
	}
	if tokens.Expiry.Before(before.Add(time.Hour - time.Minute)) {
		t.Errorf("want expiry about an hour out, got %v", tokens.Expiry)
	}
}

func TestRefreshAccessTokenReportsReconnectOnInvalidGrant(t *testing.T) {
	client := newTestSpotify(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(t, w, map[string]any{"error": "invalid_grant"})
	})

	_, err := client.RefreshAccessToken(context.Background(), "revoked")
	if !errors.Is(err, domain.ErrSpotifyReconnectRequired) {
		t.Fatalf("want ErrSpotifyReconnectRequired, got %v", err)
	}
}
