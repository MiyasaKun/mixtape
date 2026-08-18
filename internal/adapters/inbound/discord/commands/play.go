package commands

import (
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"

	"github.com/bwmarrin/discordgo"
)

type PlayCommand struct {
	musicPlayback *services.MusicPlaybackService
}

func NewPlayCommand(musicPlayback *services.MusicPlaybackService) *PlayCommand {
	return &PlayCommand{
		musicPlayback: musicPlayback,
	}
}

func (c *PlayCommand) Name() string {
	return "play"
}

func (c *PlayCommand) Description() string {
	return "Play a track from YouTube, Spotify, or SoundCloud"
}

func (c *PlayCommand) Options() []discord.CommandOption {
	return []discord.CommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "url",
			Description: "Track URL or search query",
			Required:    true,
		},
	}
}

func (c *PlayCommand) Handle(ctx *discord.CommandContext) error {
	url := ctx.GetString("url")
	guildID := ctx.GuildID()

	if err := c.musicPlayback.Play(ctx.Ctx, guildID, url); err != nil {
		return ctx.Respond("❌ Failed to play: " + err.Error())
	}

	return ctx.Respond("🎵 Now playing: " + url)
}
