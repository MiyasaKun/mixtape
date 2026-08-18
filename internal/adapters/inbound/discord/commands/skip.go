package commands

import (
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
)

type SkipCommand struct {
	musicPlayback *services.MusicPlaybackService
}

func NewSkipCommand(musicPlayback *services.MusicPlaybackService) *SkipCommand {
	return &SkipCommand{
		musicPlayback: musicPlayback,
	}
}

func (c *SkipCommand) Name() string {
	return "skip"
}

func (c *SkipCommand) Description() string {
	return "Skip the current track"
}

func (c *SkipCommand) Options() []discord.CommandOption {
	return nil
}

func (c *SkipCommand) Handle(ctx *discord.CommandContext) error {
	guildID := ctx.GuildID()

	if err := c.musicPlayback.Skip(ctx.Ctx, guildID); err != nil {
		return ctx.Respond("❌ Failed to skip: " + err.Error())
	}

	return ctx.Respond("⏭️ Skipped to next track")
}
