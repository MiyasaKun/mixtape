package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	authhandlers "fiber_test/internal/adapters/inbound/http/handlers"
	"fiber_test/internal/adapters/inbound/http/middleware"
	"fiber_test/internal/application/ports"
	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type stubVoiceGuilds struct {
	guilds []ports.VoiceGuild
}

func (s stubVoiceGuilds) ActiveVoiceGuilds() []ports.VoiceGuild {
	return s.guilds
}

type stubWebPlayback struct {
	result   *services.PlaybackResult
	err      error
	gotGuild string
	gotURL   string
	calls    int
}

func (s *stubWebPlayback) EnqueueSpotify(ctx context.Context, guildID, spotifyURL string) (*services.PlaybackResult, error) {
	s.calls++
	s.gotGuild, s.gotURL = guildID, spotifyURL
	return s.result, s.err
}

func newPlaybackApp(t *testing.T, playback *stubWebPlayback, guilds stubVoiceGuilds) (*fiber.App, string) {
	t.Helper()

	users := &stubUsers{byGoogle: map[string]domain.User{}, byID: map[string]domain.User{}}
	sessions := &stubSessions{byID: map[string]domain.Session{}}
	authSvc := services.NewAuthService(users, sessions, stubIdentities{}, time.Now, time.Hour)
	h := authhandlers.NewPlaybackHandler(playback, guilds)

	session, err := authSvc.SignInWithGoogle(context.Background(), "code", "r")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}

	app := fiber.New()
	app.Get("/api/guilds/active", middleware.RequireAuth(authSvc), h.ActiveGuilds)
	app.Post("/api/guilds/:guildId/playback/play", middleware.RequireAuth(authSvc), h.PlaySpotify)
	return app, session.ID
}

func TestActiveGuildsRequiresSession(t *testing.T) {
	app, _ := newPlaybackApp(t, &stubWebPlayback{}, stubVoiceGuilds{})

	resp := do(t, app, http.MethodGet, "/api/guilds/active", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

func TestActiveGuildsListsOnlyVoiceConnections(t *testing.T) {
	guilds := stubVoiceGuilds{guilds: []ports.VoiceGuild{{ID: "g-1", Name: "Studio"}}}
	app, session := newPlaybackApp(t, &stubWebPlayback{}, guilds)

	resp := do(t, app, http.MethodGet, "/api/guilds/active",
		map[string]string{authhandlers.SessionCookieName: session}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Guilds []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"guilds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Guilds) != 1 || payload.Guilds[0].ID != "g-1" || payload.Guilds[0].Name != "Studio" {
		t.Fatalf("unexpected guilds: %+v", payload.Guilds)
	}
}

func TestPlaySpotifyRejectsInactiveGuild(t *testing.T) {
	playback := &stubWebPlayback{}
	app, session := newPlaybackApp(t, playback, stubVoiceGuilds{})

	resp := do(t, app, http.MethodPost, "/api/guilds/g-1/playback/play",
		map[string]string{authhandlers.SessionCookieName: session},
		strings.NewReader(`{"url":"https://open.spotify.com/track/abc"}`))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d", resp.StatusCode)
	}
	if playback.calls != 0 {
		t.Error("playback must not run for an inactive guild")
	}
}

func TestPlaySpotifySendsTrackToActiveGuild(t *testing.T) {
	playback := &stubWebPlayback{result: &services.PlaybackResult{
		Track:    &services.Track{Title: "Nightcall"},
		Started:  true,
		Position: 1,
	}}
	app, session := newPlaybackApp(t, playback, stubVoiceGuilds{guilds: []ports.VoiceGuild{{ID: "g-1", Name: "Studio"}}})

	resp := do(t, app, http.MethodPost, "/api/guilds/g-1/playback/play",
		map[string]string{authhandlers.SessionCookieName: session},
		strings.NewReader(`{"url":"https://open.spotify.com/track/abc"}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	if playback.gotGuild != "g-1" || playback.gotURL != "https://open.spotify.com/track/abc" {
		t.Errorf("unexpected playback call: %+v", playback)
	}

	var payload struct {
		Status   string `json:"status"`
		Title    string `json:"title"`
		Started  bool   `json:"started"`
		Position int    `json:"position"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != "ok" || payload.Title != "Nightcall" || !payload.Started || payload.Position != 1 {
		t.Errorf("unexpected response: %+v", payload)
	}
}

func TestPlaySpotifyMapsPlaybackErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{name: "no voice connection", err: services.ErrVoiceNotConnected, wantStatus: http.StatusConflict, wantError: "bot_not_connected"},
		{name: "no match", err: services.ErrNoPlayableMatch, wantStatus: http.StatusUnprocessableEntity, wantError: "no_playable_match"},
		{name: "upstream", err: errors.New("spotify down"), wantStatus: http.StatusBadGateway, wantError: "playback_failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			playback := &stubWebPlayback{err: tc.err}
			app, session := newPlaybackApp(t, playback, stubVoiceGuilds{guilds: []ports.VoiceGuild{{ID: "g-1"}}})

			resp := do(t, app, http.MethodPost, "/api/guilds/g-1/playback/play",
				map[string]string{authhandlers.SessionCookieName: session},
				strings.NewReader(`{"url":"https://open.spotify.com/track/abc"}`))
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			var payload struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if payload.Error != tc.wantError {
				t.Errorf("want %q, got %q", tc.wantError, payload.Error)
			}
		})
	}
}
