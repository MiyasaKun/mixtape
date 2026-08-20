package commands

import (
	"fmt"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"

	dgo "github.com/disgoorg/disgo/discord"
)

type VolumeCommand struct {
	musicPlayback *services.MusicPlaybackService
}

func NewVolumeCommand(musicPlayback *services.MusicPlaybackService) *VolumeCommand {
	return &VolumeCommand{
		musicPlayback: musicPlayback,
	}
}

func (c *VolumeCommand) Name() string {
	return "volume"
}

func (c *VolumeCommand) Description() string {
	return "Set the playback volume (0-100)"
}

func (c *VolumeCommand) Options() []discord.CommandOption {
	return []discord.CommandOption{
		{
			Type:        dgo.ApplicationCommandOptionTypeInt,
			Name:        "level",
			Description: "Volume level (0-100)",
			Required:    true,
		},
	}
}

func (c *VolumeCommand) Handle(ctx *discord.CommandContext) error {
	guildID := ctx.GuildID()
	volume := ctx.GetInt("level")

	if volume < 0 || volume > 100 {
		return ctx.Respond("❌ Volume must be between 0 and 100")
	}

	if err := c.musicPlayback.SetVolume(ctx.Ctx, guildID, volume); err != nil {
		return ctx.Respond("❌ Failed to set volume: " + err.Error())
	}

	return ctx.Respond(fmt.Sprintf("🔊 Volume set to %d", volume))
}
