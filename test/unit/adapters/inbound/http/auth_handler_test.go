package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authhandlers "fiber_test/internal/adapters/inbound/http/handlers"
	"fiber_test/internal/adapters/inbound/http/middleware"
	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

// --- fakes ---

type stubUsers struct {
	byGoogle map[string]domain.User
	byID     map[string]domain.User
}

func (s *stubUsers) UpsertByGoogleID(ctx context.Context, profile domain.GoogleProfile) (domain.User, error) {
	u, ok := s.byGoogle[profile.GoogleID]
	if !ok {
		u = domain.User{ID: "u-" + profile.GoogleID, GoogleID: profile.GoogleID, Email: profile.Email, Name: profile.Name, Picture: profile.Picture}
		s.byGoogle[profile.GoogleID] = u
		s.byID[u.ID] = u
	}
	return u, nil
}

func (s *stubUsers) ByID(ctx context.Context, id string) (domain.User, error) {
	u, ok := s.byID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

type stubSessions struct {
	byID map[string]domain.Session
}

func (s *stubSessions) Create(ctx context.Context, userID string, ttl time.Duration) (domain.Session, error) {
	id := "sess-" + userID
	sess := domain.Session{ID: id, UserID: userID, ExpiresAt: time.Now().Add(ttl)}
	if s.byID == nil {
		s.byID = map[string]domain.Session{}
	}
	s.byID[id] = sess
	return sess, nil
}

func (s *stubSessions) Get(ctx context.Context, id string) (domain.Session, error) {
	sess, ok := s.byID[id]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return sess, nil
}

func (s *stubSessions) Delete(ctx context.Context, id string) error      { delete(s.byID, id); return nil }
func (s *stubSessions) DeleteExpired(ctx context.Context) (int64, error) { return 0, nil }

type stubIdentities struct{}

func (stubIdentities) AuthURL(state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
}
func (stubIdentities) ExchangeCode(ctx context.Context, code, redirectURI string) (domain.GoogleProfile, error) {
	return domain.GoogleProfile{GoogleID: "google-1", Email: "listener@example.com", Name: "Listener", Picture: "https://example.com/photo.jpg"}, nil
}
func (stubIdentities) SignOutURL(redirectURI string) string {
	return "https://accounts.google.com/logout"
}

type stubSpotifyProvider struct{}

func (stubSpotifyProvider) AuthURL(state string) string {
	return "https://accounts.spotify.com/authorize?state=" + state
}
func (stubSpotifyProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (domain.SpotifyTokens, domain.SpotifyProfile, error) {
	return domain.SpotifyTokens{AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)},
		domain.SpotifyProfile{ID: "sp-1"}, nil
}
func (stubSpotifyProvider) RefreshAccessToken(ctx context.Context, refreshToken string) (domain.SpotifyTokens, error) {
	return domain.SpotifyTokens{AccessToken: "a", Expiry: time.Now().Add(time.Hour)}, nil
}
func (stubSpotifyProvider) ListPlaylists(ctx context.Context, accessToken, spotifyUserID string, offset, limit int) (domain.PlaylistPage, error) {
	return domain.PlaylistPage{}, nil
}
func (stubSpotifyProvider) ListLikedTracks(ctx context.Context, accessToken string, offset, limit int) (domain.TrackPage, error) {
	return domain.TrackPage{}, nil
}
func (stubSpotifyProvider) ListPlaylistTracks(ctx context.Context, accessToken, playlistID string, offset, limit int) (domain.TrackPage, error) {
	return domain.TrackPage{}, nil
}

type stubStore struct {
	saved map[string]domain.SpotifyConnection
}

func (s *stubStore) Save(ctx context.Context, c domain.SpotifyConnection) error {
	if s.saved == nil {
		s.saved = map[string]domain.SpotifyConnection{}
	}
	s.saved[c.UserID] = c
	return nil
}
func (s *stubStore) ByUserID(ctx context.Context, userID string) (domain.SpotifyConnection, error) {
	c, ok := s.saved[userID]
	if !ok {
		return domain.SpotifyConnection{}, domain.ErrUserNotFound
	}
	return c, nil
}
func (s *stubStore) Delete(ctx context.Context, userID string) error {
	delete(s.saved, userID)
	return nil
}

type stubEncryptor struct{}

func (stubEncryptor) Encrypt(p string) (string, error) { return "enc:" + p, nil }
func (stubEncryptor) Decrypt(c string) (string, error) { return strings.TrimPrefix(c, "enc:"), nil }

// --- harness ---

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()

	users := &stubUsers{byGoogle: map[string]domain.User{}, byID: map[string]domain.User{}}
	sessions := &stubSessions{byID: map[string]domain.Session{}}
	authSvc := services.NewAuthService(users, sessions, stubIdentities{}, time.Now, 30*24*time.Hour)
	spotifySvc := services.NewSpotifyIntegrationService(stubSpotifyProvider{}, &stubStore{}, stubEncryptor{}, time.Now)
	h := authhandlers.NewAuthHandler(authSvc, spotifySvc, "http://localhost:3000/api/auth/google/callback", "http://localhost:3000/api/auth/spotify/callback", false)

	app := fiber.New()
	app.Get("/api/auth/google/url", h.GoogleURL)
	app.Get("/api/auth/google/callback", h.GoogleCallback)
	app.Post("/api/auth/logout", h.Logout)
	app.Get("/api/auth/me", middleware.RequireAuth(authSvc), h.Me)
	return app
}

func do(t *testing.T, app *fiber.App, method, target string, cookies map[string]string, body io.Reader) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func cookieValue(t *testing.T, resp *http.Response, name string) string {
	t.Helper()
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// --- tests ---

func TestGoogleURLReturnsConsentURLOtherwiseUnauthenticated(t *testing.T) {
	app := newTestApp(t)
	resp := do(t, app, http.MethodGet, "/api/auth/google/url", nil, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(payload.URL, "https://accounts.google.com/o/oauth2/v2/auth") {
		t.Errorf("unexpected url %q", payload.URL)
	}
	if cookieValue(t, resp, authhandlers.GoogleStateCookie) == "" {
		t.Error("want a state cookie to be set")
	}
}

func TestGoogleCallbackRejectsStateMismatch(t *testing.T) {
	app := newTestApp(t)
	resp := do(t, app, http.MethodGet, "/api/auth/google/callback?code=abc&state=wrong",
		map[string]string{authhandlers.GoogleStateCookie: "expected"}, nil)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
	if cookieValue(t, resp, authhandlers.SessionCookieName) != "" {
		t.Error("no session cookie should be issued on a rejected callback")
	}
}

func TestGoogleCallbackIssuesSessionAndRedirects(t *testing.T) {
	app := newTestApp(t)
	resp := do(t, app, http.MethodGet, "/api/auth/google/callback?code=abc&state=match",
		map[string]string{authhandlers.GoogleStateCookie: "match"}, nil)

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/dashboard" {
		t.Errorf("want redirect to /dashboard, got %q", loc)
	}
	if cookieValue(t, resp, authhandlers.SessionCookieName) == "" {
		t.Error("want a session cookie")
	}
}

func TestMeRequiresSession(t *testing.T) {
	app := newTestApp(t)
	resp := do(t, app, http.MethodGet, "/api/auth/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without cookie, got %d", resp.StatusCode)
	}
}

func TestMeReturnsSignedInAccount(t *testing.T) {
	app := newTestApp(t)
	resp := do(t, app, http.MethodGet, "/api/auth/google/callback?code=abc&state=match",
		map[string]string{authhandlers.GoogleStateCookie: "match"}, nil)
	session := cookieValue(t, resp, authhandlers.SessionCookieName)
	if session == "" {
		t.Fatal("no session cookie from callback")
	}

	me := do(t, app, http.MethodGet, "/api/auth/me",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", me.StatusCode)
	}

	var payload struct {
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(me.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Email != "listener@example.com" {
		t.Errorf("want listener@example.com, got %q", payload.Email)
	}
	if payload.Picture != "https://example.com/photo.jpg" {
		t.Errorf("want the Google picture, got %q", payload.Picture)
	}
}
