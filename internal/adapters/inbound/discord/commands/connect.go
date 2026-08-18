package commands

import (
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"

	"github.com/bwmarrin/discordgo"
)

type ConnectCommand struct {
	guildConfig *services.GuildConfigService
}

func NewConnectCommand(guildConfig *services.GuildConfigService) *ConnectCommand {
	return &ConnectCommand{
		guildConfig: guildConfig,
	}
}

func (c *ConnectCommand) Name() string {
	return "connect"
}

func (c *ConnectCommand) Description() string {
	return "Connect the bot to a voice channel"
}

func (c *ConnectCommand) Options() []discord.CommandOption {
	return []discord.CommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionChannel,
			Name:        "channel",
			Description: "Voice channel to connect to",
			Required:    true,
		},
	}
}

func (c *ConnectCommand) Handle(ctx *discord.CommandContext) error {
	guildID := ctx.GuildID()
	channelID := ctx.GetString("channel")

	if err := c.guildConfig.SetVoiceChannel(ctx.Ctx, guildID, channelID); err != nil {
		return ctx.Respond("❌ Failed to connect: " + err.Error())
	}

	return ctx.Respond("✅ Connected to voice channel")
}
