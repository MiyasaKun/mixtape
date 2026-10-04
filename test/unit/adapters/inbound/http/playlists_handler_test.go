package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	authhandlers "fiber_test/internal/adapters/inbound/http/handlers"
	"fiber_test/internal/adapters/inbound/http/middleware"
	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type playlistProvider struct {
	playlists     domain.PlaylistPage
	liked         domain.TrackPage
	tracks        domain.TrackPage
	err           error
	gotPlaylistID string
	gotOffset     int
	gotLimit      int
}

func (*playlistProvider) ExchangeCode(context.Context, string, string) (domain.SpotifyTokens, domain.SpotifyProfile, error) {
	return domain.SpotifyTokens{}, domain.SpotifyProfile{}, nil
}
func (*playlistProvider) AuthURL(string) string { return "" }
func (p *playlistProvider) RefreshAccessToken(context.Context, string) (domain.SpotifyTokens, error) {
	return domain.SpotifyTokens{}, nil
}
func (p *playlistProvider) ListPlaylists(_ context.Context, _ string, _ string, offset, limit int) (domain.PlaylistPage, error) {
	p.gotOffset, p.gotLimit = offset, limit
	return p.playlists, p.err
}
func (p *playlistProvider) ListLikedTracks(_ context.Context, _ string, offset, limit int) (domain.TrackPage, error) {
	p.gotOffset, p.gotLimit = offset, limit
	return p.liked, p.err
}
func (p *playlistProvider) ListPlaylistTracks(_ context.Context, _ string, playlistID string, offset, limit int) (domain.TrackPage, error) {
	p.gotPlaylistID, p.gotOffset, p.gotLimit = playlistID, offset, limit
	return p.tracks, p.err
}

func connectedConn(userID string) domain.SpotifyConnection {
	return domain.SpotifyConnection{
		UserID:             userID,
		SpotifyUserID:      "sp-1",
		AccessToken:        "access",
		RefreshTokenCipher: "enc:refresh",
		TokenExpiry:        time.Now().Add(time.Hour),
	}
}

func newPlaylistApp(t *testing.T, provider *playlistProvider, store *stubStore) (*fiber.App, string) {
	t.Helper()

	users := &stubUsers{byGoogle: map[string]domain.User{}, byID: map[string]domain.User{}}
	sessions := &stubSessions{byID: map[string]domain.Session{}}
	authSvc := services.NewAuthService(users, sessions, stubIdentities{}, time.Now, time.Hour)
	spotifySvc := services.NewSpotifyIntegrationService(provider, store, stubEncryptor{}, time.Now)
	h := authhandlers.NewPlaylistHandler(spotifySvc)

	session, err := authSvc.SignInWithGoogle(context.Background(), "code", "r")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}

	app := fiber.New()
	app.Get("/api/spotify/playlists", middleware.RequireAuth(authSvc), h.ListPlaylists)
	app.Get("/api/spotify/playlists/:id/tracks", middleware.RequireAuth(authSvc), h.ListPlaylistTracks)
	app.Get("/api/spotify/liked", middleware.RequireAuth(authSvc), h.ListLikedTracks)
	return app, session.ID
}

func TestPlaylistRoutesRequireSession(t *testing.T) {
	app, _ := newPlaylistApp(t, &playlistProvider{}, &stubStore{})

	for _, target := range []string{"/api/spotify/playlists", "/api/spotify/playlists/pl-1/tracks", "/api/spotify/liked"} {
		resp := do(t, app, http.MethodGet, target, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: want 401, got %d", target, resp.StatusCode)
		}
	}
}

func TestPlaylistsRequireSpotifyConnection(t *testing.T) {
	app, session := newPlaylistApp(t, &playlistProvider{}, &stubStore{})
	resp := do(t, app, http.MethodGet, "/api/spotify/playlists",
		map[string]string{authhandlers.SessionCookieName: session}, nil)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d", resp.StatusCode)
	}
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Error != "spotify_not_connected" {
		t.Errorf("want spotify_not_connected, got %q", payload.Error)
	}
}

func TestPlaylistsReturnNormalizedPage(t *testing.T) {
	provider := &playlistProvider{playlists: domain.PlaylistPage{
		Items: []domain.Playlist{{ID: "pl-1", Name: "Focus", Owner: "Ada", TrackCount: 12, ImageURL: "https://img/cover.jpg", URL: "https://open.spotify.com/playlist/pl-1", IsOwned: true, Collaborative: true, Browseable: true}},
	}}
	store := &stubStore{saved: map[string]domain.SpotifyConnection{"u-google-1": connectedConn("u-google-1")}}
	app, session := newPlaylistApp(t, provider, store)

	resp := do(t, app, http.MethodGet, "/api/spotify/playlists?offset=0&limit=50",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Items []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Owner         string `json:"owner"`
			TrackCount    int    `json:"trackCount"`
			ImageURL      string `json:"imageUrl"`
			IsOwned       bool   `json:"isOwned"`
			Collaborative bool   `json:"collaborative"`
			Browseable    bool   `json:"browseable"`
		} `json:"items"`
		Next *int `json:"next"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("want 1 playlist, got %d", len(payload.Items))
	}
	got := payload.Items[0]
	if got.ID != "pl-1" || got.Name != "Focus" || got.Owner != "Ada" || got.TrackCount != 12 {
		t.Errorf("unexpected playlist: %+v", got)
	}
	if !got.IsOwned || !got.Collaborative || !got.Browseable || got.ImageURL != "https://img/cover.jpg" {
		t.Errorf("unexpected flags: %+v", got)
	}
	if payload.Next != nil {
		t.Errorf("want next null, got %v", *payload.Next)
	}
}

func TestPlaylistsPassPaginationToProvider(t *testing.T) {
	provider := &playlistProvider{}
	store := &stubStore{saved: map[string]domain.SpotifyConnection{"u-google-1": connectedConn("u-google-1")}}
	app, session := newPlaylistApp(t, provider, store)

	resp := do(t, app, http.MethodGet, "/api/spotify/playlists?offset=100&limit=999",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	if provider.gotOffset != 100 {
		t.Errorf("want offset 100, got %d", provider.gotOffset)
	}
	if provider.gotLimit != 50 {
		t.Errorf("want limit clamped to 50, got %d", provider.gotLimit)
	}
}

func TestLikedTracksReturnNormalizedTracks(t *testing.T) {
	provider := &playlistProvider{liked: domain.TrackPage{
		Items: []domain.PlaylistTrack{{ID: "tr-1", Title: "Nightcall", Artists: []string{"Kavinsky"}, Album: "OutRun", DurationMS: 258000, ImageURL: "https://img/a.jpg", URL: "https://open.spotify.com/track/tr-1"}},
	}}
	store := &stubStore{saved: map[string]domain.SpotifyConnection{"u-google-1": connectedConn("u-google-1")}}
	app, session := newPlaylistApp(t, provider, store)

	resp := do(t, app, http.MethodGet, "/api/spotify/liked",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Items []struct {
			ID         string   `json:"id"`
			Title      string   `json:"title"`
			Artists    []string `json:"artists"`
			Album      string   `json:"album"`
			DurationMS int      `json:"durationMs"`
			ImageURL   string   `json:"imageUrl"`
			URL        string   `json:"url"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("want 1 track, got %d", len(payload.Items))
	}
	got := payload.Items[0]
	if got.ID != "tr-1" || got.Title != "Nightcall" || got.DurationMS != 258000 {
		t.Errorf("unexpected track: %+v", got)
	}
	if len(got.Artists) != 1 || got.Artists[0] != "Kavinsky" {
		t.Errorf("want artists, got %v", got.Artists)
	}
}

func TestPlaylistTracksForwardPlaylistID(t *testing.T) {
	provider := &playlistProvider{tracks: domain.TrackPage{Items: []domain.PlaylistTrack{{ID: "tr-9", Title: "Resonance"}}}}
	store := &stubStore{saved: map[string]domain.SpotifyConnection{"u-google-1": connectedConn("u-google-1")}}
	app, session := newPlaylistApp(t, provider, store)

	resp := do(t, app, http.MethodGet, "/api/spotify/playlists/pl-9/tracks",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	if provider.gotPlaylistID != "pl-9" {
		t.Errorf("want playlist id pl-9, got %q", provider.gotPlaylistID)
	}
}

func TestForbiddenPlaylistReturnsForbidden(t *testing.T) {
	provider := &playlistProvider{err: &domain.SpotifyAPIError{StatusCode: http.StatusForbidden}}
	store := &stubStore{saved: map[string]domain.SpotifyConnection{"u-google-1": connectedConn("u-google-1")}}
	app, session := newPlaylistApp(t, provider, store)

	resp := do(t, app, http.MethodGet, "/api/spotify/playlists/pl-9/tracks",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Error != "spotify_forbidden" {
		t.Errorf("want spotify_forbidden, got %q", payload.Error)
	}
}

func TestReconnectRequiredReturnsConflict(t *testing.T) {
	provider := &playlistProvider{err: domain.ErrSpotifyReconnectRequired}
	store := &stubStore{saved: map[string]domain.SpotifyConnection{"u-google-1": connectedConn("u-google-1")}}
	app, session := newPlaylistApp(t, provider, store)

	resp := do(t, app, http.MethodGet, "/api/spotify/playlists",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d", resp.StatusCode)
	}
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Error != "spotify_reconnect_required" {
		t.Errorf("want spotify_reconnect_required, got %q", payload.Error)
	}
}
