package handlers

import (
	"encoding/json"
	"time"

	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

const (
	// SessionCookieName carries the opaque session ID.
	SessionCookieName = "mixtape_session"
	// GoogleStateCookie holds the anti-CSRF state for the Google flow.
	GoogleStateCookie = "mixtape_google_state"
	// SpotifyStateCookie holds the anti-CSRF state for the Spotify flow.
	SpotifyStateCookie = "mixtape_spotify_state"
	// UserLocalsKey is where RequireAuth stores the signed-in account.
	UserLocalsKey = "mixtape_user"
)

// AuthHandler serves the sign-in, session, and Spotify-connection routes.
type AuthHandler struct {
	auth               *services.AuthService
	spotify            *services.SpotifyIntegrationService
	googleRedirectURI  string
	spotifyRedirectURI string
	secureCookies      bool
}

func NewAuthHandler(
	auth *services.AuthService,
	spotify *services.SpotifyIntegrationService,
	googleRedirectURI, spotifyRedirectURI string,
	secureCookies bool,
) *AuthHandler {
	return &AuthHandler{
		auth:               auth,
		spotify:            spotify,
		googleRedirectURI:  googleRedirectURI,
		spotifyRedirectURI: spotifyRedirectURI,
		secureCookies:      secureCookies,
	}
}

// GoogleURL returns the Google consent URL and plants the state cookie.
func (h *AuthHandler) GoogleURL(c fiber.Ctx) error {
	state, err := newState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start sign-in"})
	}
	h.setStateCookie(c, GoogleStateCookie, state)
	return c.JSON(fiber.Map{"url": h.auth.AuthURL(state)})
}

// GoogleCallback completes sign-in and issues the session cookie.
func (h *AuthHandler) GoogleCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" || state != c.Cookies(GoogleStateCookie) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sign-in callback"})
	}
	h.clearCookie(c, GoogleStateCookie)

	session, err := h.auth.SignInWithGoogle(c.Context(), code, h.googleRedirectURI)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "sign-in failed"})
	}

	h.setSessionCookie(c, session)
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard")
}

// Me returns the signed-in account.
func (h *AuthHandler) Me(c fiber.Ctx) error {
	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(userResponse(user))
}

type updatePreferencesRequest struct {
	SpotifyAutoplay *bool `json:"spotifyAutoplay"`
}

// UpdatePreferences updates the signed-in account's web preferences.
func (h *AuthHandler) UpdatePreferences(c fiber.Ctx) error {
	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body updatePreferencesRequest
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.SpotifyAutoplay == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	if err := h.auth.SetSpotifyAutoplay(c.Context(), user.ID, *body.SpotifyAutoplay); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "update_failed"})
	}

	user.SpotifyAutoplay = *body.SpotifyAutoplay
	return c.JSON(userResponse(user))
}

// Logout revokes the session and clears the cookie.
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	if sid := c.Cookies(SessionCookieName); sid != "" {
		if err := h.auth.SignOut(c.Context(), sid); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "sign-out failed"})
		}
	}
	h.clearCookie(c, SessionCookieName)
	return c.JSON(fiber.Map{"status": "ok"})
}

// SpotifyURL returns the Spotify consent URL and plants the state cookie.
func (h *AuthHandler) SpotifyURL(c fiber.Ctx) error {
	state, err := newState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start Spotify connection"})
	}
	h.setStateCookie(c, SpotifyStateCookie, state)
	return c.JSON(fiber.Map{"url": h.spotify.AuthURL(state)})
}

// SpotifyCallback stores the Spotify link for the signed-in user.
func (h *AuthHandler) SpotifyCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" || state != c.Cookies(SpotifyStateCookie) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid Spotify callback"})
	}
	h.clearCookie(c, SpotifyStateCookie)

	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.spotify.Connect(c.Context(), user.ID, code, h.spotifyRedirectURI); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "could not connect Spotify"})
	}
	return c.Redirect().Status(fiber.StatusFound).To("/settings")
}

// SpotifyDisconnect removes the signed-in user's Spotify link.
func (h *AuthHandler) SpotifyDisconnect(c fiber.Ctx) error {
	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if err := h.spotify.Disconnect(c.Context(), user.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not disconnect Spotify"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *AuthHandler) setSessionCookie(c fiber.Ctx, session domain.Session) {
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		HTTPOnly: true,
		Secure:   h.secureCookies,
		SameSite: "Lax",
	})
}

func (h *AuthHandler) setStateCookie(c fiber.Ctx, name, state string) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    state,
		Path:     "/",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HTTPOnly: true,
		Secure:   h.secureCookies,
		SameSite: "Lax",
	})
}

func (h *AuthHandler) clearCookie(c fiber.Ctx, name string) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   h.secureCookies,
		SameSite: "Lax",
	})
}

func userResponse(u domain.User) fiber.Map {
	return fiber.Map{
		"id":               u.ID,
		"email":            u.Email,
		"name":             u.Name,
		"picture":          u.Picture,
		"locale":           u.Locale,
		"spotifyConnected": u.SpotifyConnected,
		"spotifyAutoplay":  u.SpotifyAutoplay,
	}
}
