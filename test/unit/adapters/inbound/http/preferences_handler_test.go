package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	authhandlers "fiber_test/internal/adapters/inbound/http/handlers"
	"fiber_test/internal/adapters/inbound/http/middleware"
	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

func newPreferencesApp(t *testing.T) (*fiber.App, string) {
	t.Helper()

	users := &stubUsers{byGoogle: map[string]domain.User{}, byID: map[string]domain.User{}}
	sessions := &stubSessions{byID: map[string]domain.Session{}}
	authSvc := services.NewAuthService(users, sessions, stubIdentities{}, time.Now, time.Hour)
	h := authhandlers.NewAuthHandler(authSvc, nil, "", "", false)

	session, err := authSvc.SignInWithGoogle(context.Background(), "code", "r")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}

	app := fiber.New()
	app.Get("/api/auth/me", middleware.RequireAuth(authSvc), h.Me)
	app.Put("/api/auth/me/preferences", middleware.RequireAuth(authSvc), h.UpdatePreferences)
	return app, session.ID
}

func TestUpdatePreferencesPersistsSpotifyAutoplay(t *testing.T) {
	app, session := newPreferencesApp(t)

	resp := do(t, app, http.MethodPut, "/api/auth/me/preferences",
		map[string]string{authhandlers.SessionCookieName: session},
		strings.NewReader(`{"spotifyAutoplay":true}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var updated struct {
		SpotifyAutoplay bool `json:"spotifyAutoplay"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !updated.SpotifyAutoplay {
		t.Error("want spotifyAutoplay true in the response")
	}

	me := do(t, app, http.MethodGet, "/api/auth/me",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("want 200 from /me, got %d", me.StatusCode)
	}
	var user struct {
		SpotifyAutoplay bool `json:"spotifyAutoplay"`
	}
	if err := json.NewDecoder(me.Body).Decode(&user); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if !user.SpotifyAutoplay {
		t.Error("preference must survive the follow-up read")
	}
}

func TestUpdatePreferencesRejectsMissingField(t *testing.T) {
	app, session := newPreferencesApp(t)

	resp := do(t, app, http.MethodPut, "/api/auth/me/preferences",
		map[string]string{authhandlers.SessionCookieName: session},
		strings.NewReader(`{}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

func TestUpdatePreferencesRequiresSession(t *testing.T) {
	app, _ := newPreferencesApp(t)

	resp := do(t, app, http.MethodPut, "/api/auth/me/preferences", nil,
		strings.NewReader(`{"spotifyAutoplay":true}`))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}
