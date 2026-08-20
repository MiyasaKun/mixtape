package commands

import (
	"context"
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
	"time"

	dgo "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
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
			Type:        dgo.ApplicationCommandOptionTypeChannel,
			Name:        "channel",
			Description: "Voice channel to connect to",
			Required:    true,
		},
	}
}

func (c *ConnectCommand) Handle(ctx *discord.CommandContext) error {
	// Defer for more time to run functions
	err := ctx.Defer()

	if err != nil {
		return err
	}

	guildID := ctx.GuildID()
	parsedGuildId, _ := snowflake.Parse(guildID)
	channelID := ctx.GetString("channel")
	parsedChannelId, _ := snowflake.Parse(channelID)

	// Generate a Connection to connect against
	conn := ctx.Client.VoiceManager.CreateConn(parsedGuildId)
	openCtx, cancel := context.WithTimeout(ctx.Ctx, time.Second*5)

	defer cancel()

	//Connect to Voice Channel
	err = conn.Open(openCtx, parsedChannelId, false, false)

	if err != nil {
		conn.Close(context.Background())
		return ctx.EditResponse("Error while connecting to the voice channel")
	}

	return ctx.EditResponse("✅ Connected to voice channel")
}
