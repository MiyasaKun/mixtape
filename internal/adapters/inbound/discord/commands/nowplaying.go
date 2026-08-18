package commands

import (
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
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

	track, err := c.musicPlayback.GetCurrentTrack(ctx.Ctx, guildID)
	if err != nil {
		return ctx.Respond("❌ Failed to get current track: " + err.Error())
	}

	if track == nil {
		return ctx.Respond("📭 Nothing is playing right now")
	}

	description := "**" + track.Title + "**"
	if track.Artist != "" {
		description += " by " + track.Artist
	}

	return ctx.RespondEmbed("🎵 Now Playing", description)
}
