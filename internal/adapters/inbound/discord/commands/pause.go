package commands

import (
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
)

type PauseCommand struct {
	musicPlayback *services.MusicPlaybackService
}

func NewPauseCommand(musicPlayback *services.MusicPlaybackService) *PauseCommand {
	return &PauseCommand{
		musicPlayback: musicPlayback,
	}
}

func (c *PauseCommand) Name() string {
	return "pause"
}

func (c *PauseCommand) Description() string {
	return "Pause the current playback"
}

func (c *PauseCommand) Options() []discord.CommandOption {
	return nil
}

func (c *PauseCommand) Handle(ctx *discord.CommandContext) error {
	guildID := ctx.GuildID()

	if err := c.musicPlayback.Pause(ctx.Ctx, guildID); err != nil {
		return ctx.Respond("❌ Failed to pause: " + err.Error())
	}

	return ctx.Respond("⏸️ Playback paused")
}
