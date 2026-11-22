package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"

	"github.com/gofiber/fiber/v3"
)

// WebPlayback sends a track selected in the web UI to the bot.
type WebPlayback interface {
	EnqueueSpotify(ctx context.Context, guildID, spotifyURL string) (*services.PlaybackResult, error)
}

// PlaybackHandler serves the web-to-Discord playback routes. Every route
// requires a signed-in user; the guild must be one where the bot currently
// has an active voice connection.
type PlaybackHandler struct {
	playback WebPlayback
	guilds   ports.VoiceGuildLister
}

func NewPlaybackHandler(playback WebPlayback, guilds ports.VoiceGuildLister) *PlaybackHandler {
	return &PlaybackHandler{playback: playback, guilds: guilds}
}

type activeGuildResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ActiveGuilds lists the guilds the bot is currently connected to in voice.
func (h *PlaybackHandler) ActiveGuilds(c fiber.Ctx) error {
	if _, ok := c.Locals(UserLocalsKey).(domain.User); !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	guilds := h.guilds.ActiveVoiceGuilds()
	items := make([]activeGuildResponse, 0, len(guilds))
	for _, guild := range guilds {
		items = append(items, activeGuildResponse{ID: guild.ID, Name: guild.Name})
	}
	return c.JSON(fiber.Map{"guilds": items})
}

type playTrackRequest struct {
	URL string `json:"url"`
}

type playTrackResponse struct {
	Status   string `json:"status"`
	Title    string `json:"title"`
	Started  bool   `json:"started"`
	Position int    `json:"position"`
}

// PlaySpotify resolves a Spotify track and sends it to the chosen guild's
// queue, starting playback when the guild is idle.
func (h *PlaybackHandler) PlaySpotify(c fiber.Ctx) error {
	if _, ok := c.Locals(UserLocalsKey).(domain.User); !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	guildID := c.Params("guildId")
	if !h.isActive(guildID) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "bot_not_connected"})
	}

	var body playTrackRequest
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	result, err := h.playback.EnqueueSpotify(c.Context(), guildID, body.URL)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrVoiceNotConnected):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "bot_not_connected"})
		case errors.Is(err, services.ErrNoPlayableMatch):
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "no_playable_match"})
		default:
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "playback_failed"})
		}
	}

	return c.JSON(playTrackResponse{
		Status:   "ok",
		Title:    result.Track.Title,
		Started:  result.Started,
		Position: result.Position,
	})
}

func (h *PlaybackHandler) isActive(guildID string) bool {
	if guildID == "" {
		return false
	}
	for _, guild := range h.guilds.ActiveVoiceGuilds() {
		if guild.ID == guildID {
			return true
		}
	}
	return false
}
