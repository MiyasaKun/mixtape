package commands

import (
	"time"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"

	dgo "github.com/disgoorg/disgo/discord"
)

type NowPlayingCommand struct {
	musicPlayback *services.MusicPlaybackService
}

func NewNowPlayingCommand(musicPlayback *services.MusicPlaybackService) *NowPlayingCommand {
	return &NowPlayingCommand{
		musicPlayback: musicPlayback,
	}
}

func (c *NowPlayingCommand) Name() string {
	return "nowplaying"
}

func (c *NowPlayingCommand) Description() string {
	return "Show the currently playing track"
}

func (c *NowPlayingCommand) Options() []discord.CommandOption {
	return nil
}

func (c *NowPlayingCommand) Handle(ctx *discord.CommandContext) error {
	guildID := ctx.GuildID()

	status, err := c.musicPlayback.GetPlaybackStatus(ctx.Ctx, guildID)
	if err != nil {
		return ctx.Respond("❌ Failed to get current track: " + err.Error())
	}

	if status == nil {
		return ctx.Respond("📭 Nothing is playing right now")
	}

	// Build the embed directly here (like play.go uses the Rest client
	// directly); the command layer only glues, the panel is rendered by the
	// pure helper in the discord package. The card is purely visual: embed
	// only, no interactive components.
	embeds := discord.BuildNowPlayingPanel(status, time.Now())
	return ctx.Interaction.Respond(dgo.InteractionResponseTypeCreateMessage, dgo.MessageCreate{
		Embeds: embeds,
	})
}
