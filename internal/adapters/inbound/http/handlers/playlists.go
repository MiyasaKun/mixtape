package handlers

import (
	"errors"
	"strconv"

	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

const (
	defaultPlaylistLimit = 50
	maxPlaylistLimit     = 50
	defaultTrackLimit    = 100
	maxTrackLimit        = 100
)

// PlaylistHandler serves the read-only Spotify playlist browsing routes. Every
// route requires a signed-in user with a Spotify connection.
type PlaylistHandler struct {
	spotify *services.SpotifyIntegrationService
}

func NewPlaylistHandler(spotify *services.SpotifyIntegrationService) *PlaylistHandler {
	return &PlaylistHandler{spotify: spotify}
}

// ListPlaylists returns a page of the user's followed and owned playlists.
func (h *PlaylistHandler) ListPlaylists(c fiber.Ctx) error {
	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	offset, limit := pagination(c, defaultPlaylistLimit, maxPlaylistLimit)
	page, err := h.spotify.ListPlaylists(c.Context(), user.ID, offset, limit)
	if err != nil {
		return spotifyError(c, err)
	}
	return c.JSON(fiber.Map{"items": playlistResponses(page.Items), "next": page.Next})
}

// ListLikedTracks returns a page of the user's Liked Songs.
func (h *PlaylistHandler) ListLikedTracks(c fiber.Ctx) error {
	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	offset, limit := pagination(c, defaultTrackLimit, maxTrackLimit)
	page, err := h.spotify.ListLikedTracks(c.Context(), user.ID, offset, limit)
	if err != nil {
		return spotifyError(c, err)
	}
	return c.JSON(fiber.Map{"items": trackResponses(page.Items), "next": page.Next})
}

// ListPlaylistTracks returns a page of tracks in a single playlist.
func (h *PlaylistHandler) ListPlaylistTracks(c fiber.Ctx) error {
	user, ok := c.Locals(UserLocalsKey).(domain.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	offset, limit := pagination(c, defaultTrackLimit, maxTrackLimit)
	page, err := h.spotify.ListPlaylistTracks(c.Context(), user.ID, c.Params("id"), offset, limit)
	if err != nil {
		return spotifyError(c, err)
	}
	return c.JSON(fiber.Map{"items": trackResponses(page.Items), "next": page.Next})
}

// spotifyError maps the Spotify service's errors to HTTP responses the frontend
// can branch on. Upstream statuses are classified first so conditions such as a
// non-readable playlist are not reported as a generic outage.
func spotifyError(c fiber.Ctx, err error) error {
	var apiErr *domain.SpotifyAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case fiber.StatusUnauthorized:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "spotify_reconnect_required"})
		case fiber.StatusForbidden:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "spotify_forbidden"})
		case fiber.StatusNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "spotify_not_found"})
		case fiber.StatusTooManyRequests:
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "spotify_rate_limited"})
		}
	}

	switch {
	case errors.Is(err, domain.ErrSpotifyNotConnected):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "spotify_not_connected"})
	case errors.Is(err, domain.ErrSpotifyReconnectRequired):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "spotify_reconnect_required"})
	default:
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "spotify_unavailable"})
	}
}

// pagination reads offset/limit query parameters, defaulting and clamping them.
func pagination(c fiber.Ctx, defaultLimit, maxLimit int) (offset, limit int) {
	offset = 0
	if raw := c.Query("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}

	limit = defaultLimit
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return offset, limit
}

type playlistResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Owner         string `json:"owner"`
	TrackCount    int    `json:"trackCount"`
	ImageURL      string `json:"imageUrl"`
	URL           string `json:"url"`
	IsOwned       bool   `json:"isOwned"`
	Collaborative bool   `json:"collaborative"`
	Browseable    bool   `json:"browseable"`
}

type trackResponse struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Artists    []string `json:"artists"`
	Album      string   `json:"album"`
	DurationMS int      `json:"durationMs"`
	ImageURL   string   `json:"imageUrl"`
	URL        string   `json:"url"`
}

func playlistResponses(items []domain.Playlist) []playlistResponse {
	out := make([]playlistResponse, 0, len(items))
	for _, p := range items {
		out = append(out, playlistResponse{
			ID:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			Owner:         p.Owner,
			TrackCount:    p.TrackCount,
			ImageURL:      p.ImageURL,
			URL:           p.URL,
			IsOwned:       p.IsOwned,
			Collaborative: p.Collaborative,
			Browseable:    p.Browseable,
		})
	}
	return out
}

func trackResponses(items []domain.PlaylistTrack) []trackResponse {
	out := make([]trackResponse, 0, len(items))
	for _, t := range items {
		artists := t.Artists
		if artists == nil {
			artists = []string{}
		}
		out = append(out, trackResponse{
			ID:         t.ID,
			Title:      t.Title,
			Artists:    artists,
			Album:      t.Album,
			DurationMS: t.DurationMS,
			ImageURL:   t.ImageURL,
			URL:        t.URL,
		})
	}
	return out
}
