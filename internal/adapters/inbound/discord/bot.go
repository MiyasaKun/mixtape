package discord

import (
	"context"

	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/logging"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/thomas-vilte/dave-go/session"
)

const guildID = ""

type Bot struct {
	client             *bot.Client
	logger             *logging.Logger
	commands           map[string]Command
	musicPlayback      *services.MusicPlaybackService
	queue              *services.QueueService
	guildConfig        *services.GuildConfigService
	spotifyIntegration *services.SpotifyIntegrationService
}

func NewBot(
	token string,
	logger *logging.Logger,
	musicPlayback *services.MusicPlaybackService,
	queue *services.QueueService,
	guildConfig *services.GuildConfigService,
	spotifyIntegration *services.SpotifyIntegrationService,
) *Bot {
	client, _ := disgo.New(token,
		bot.WithGatewayConfigOpts(gateway.WithIntents(
			gateway.IntentGuilds,
			gateway.IntentGuildVoiceStates,
		)),
		bot.WithCacheConfigOpts(cache.WithCaches(
			cache.FlagGuilds,
			cache.FlagVoiceStates,
		)),
		bot.WithEventManagerConfigOpts(bot.WithAsyncEventsEnabled()),
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(session.CreateFunc()),
		),
	)

	b := &Bot{
		client:             client,
		logger:             logger,
		commands:           make(map[string]Command),
		musicPlayback:      musicPlayback,
		queue:              queue,
		guildConfig:        guildConfig,
		spotifyIntegration: spotifyIntegration,
	}

	client.AddEventListeners(
		bot.NewListenerFunc(b.handleInteraction),
		bot.NewListenerFunc(b.handleGuildVoiceJoin),
		bot.NewListenerFunc(b.handleGuildVoiceMove),
		bot.NewListenerFunc(b.handleGuildVoiceLeave),
	)

	return b
}

func (b *Bot) RegisterCommand(cmd Command) {
	b.commands[cmd.Name()] = cmd
	b.logger.Info("registered command", "name", cmd.Name())
}

func (b *Bot) RegisterCommands(cmds ...Command) {
	for _, cmd := range cmds {
		b.RegisterCommand(cmd)
	}
}

func (b *Bot) Start() error {
	if err := b.client.OpenGateway(context.TODO()); err != nil {
		return err
	}
	b.registerDiscordCommands()

	b.logger.Info("discord bot started", "commands", len(b.commands))
	return nil
}

func (b *Bot) Stop() {
	b.client.Close(context.TODO())
}

func (b *Bot) registerDiscordCommands() {
	discordCmds := make([]discord.ApplicationCommandCreate, 0, len(b.commands))

	for _, cmd := range b.commands {
		discordCmds = append(discordCmds, commandToDiscord(cmd))
	}

	_, err := b.client.Rest.SetGlobalCommands(b.client.ApplicationID, discordCmds)
	if err != nil {
		b.logger.Error("failed to register global commands", "error", err)
	} else {
		b.logger.Info("registered global commands", "count", len(discordCmds))
	}
}

func (b *Bot) handleInteraction(e *events.ApplicationCommandInteractionCreate) {
	data := e.SlashCommandInteractionData()
	cmdName := data.CommandName()

	cmd, exists := b.commands[cmdName]
	if !exists {
		b.logger.Warn("unknown command received", "command", cmdName)
		return
	}

	ctx := &CommandContext{
		Client:      b.client,
		Interaction: e,
		Ctx:         context.Background(),
		Options:     parseOptions(e),
	}

	if err := cmd.Handle(ctx); err != nil {
		b.logger.Error("command handler failed", "command", cmdName, "error", err)
		if respondErr := ctx.Respond(formatCommandError(cmdName, err)); respondErr != nil {
			b.logger.Error("failed to send error response", "error", respondErr)
		}
	}
}

func (b *Bot) handleGuildVoiceJoin(e *events.GuildVoiceJoin) {
	if e.VoiceState.UserID == b.client.ID() {
		return
	}
	b.touchIfInBotChannel(e.VoiceState.GuildID, e.VoiceState.ChannelID)
}

func (b *Bot) handleGuildVoiceMove(e *events.GuildVoiceMove) {
	if e.VoiceState.UserID == b.client.ID() {
		return
	}
	b.touchIfInBotChannel(e.VoiceState.GuildID, e.VoiceState.ChannelID)
	b.disconnectIfAlone(e.VoiceState.GuildID)
}

func (b *Bot) handleGuildVoiceLeave(e *events.GuildVoiceLeave) {
	if e.VoiceState.UserID == b.client.ID() {
		return
	}
	b.disconnectIfAlone(e.VoiceState.GuildID)
}

func (b *Bot) disconnectIfAlone(guildID snowflake.ID) {
	conn := b.client.VoiceManager.GetConn(guildID)
	if conn == nil || conn.ChannelID() == nil {
		return
	}

	channel, ok := b.client.Caches.GuildAudioChannel(*conn.ChannelID())
	if !ok {
		return
	}

	self, ok := b.client.Caches.SelfUser()
	if !ok {
		return
	}

	for _, member := range b.client.Caches.AudioChannelMembers(channel) {
		if member.User.ID != self.ID {
			return
		}
	}

	b.musicPlayback.Disconnect(context.Background(), guildID.String())
}

func (b *Bot) touchIfInBotChannel(guildID snowflake.ID, channelID *snowflake.ID) {
	conn := b.client.VoiceManager.GetConn(guildID)
	if conn == nil || conn.ChannelID() == nil || channelID == nil || *conn.ChannelID() != *channelID {
		return
	}

	b.musicPlayback.Touch(guildID.String())
}
