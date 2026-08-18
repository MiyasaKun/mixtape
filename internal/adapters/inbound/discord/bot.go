package discord

import (
	"context"

	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/logging"

	"github.com/bwmarrin/discordgo"
)

const guildID = ""

type Bot struct {
	session         *discordgo.Session
	logger          *logging.Logger
	commands        map[string]Command
	musicPlayback   *services.MusicPlaybackService
	queue           *services.QueueService
	guildConfig     *services.GuildConfigService
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
	session, _ := discordgo.New("Bot " + token)

	bot := &Bot{
		session:            session,
		logger:             logger,
		commands:           make(map[string]Command),
		musicPlayback:      musicPlayback,
		queue:              queue,
		guildConfig:        guildConfig,
		spotifyIntegration: spotifyIntegration,
	}

	session.AddHandler(bot.handleInteraction)

	return bot
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
	if err := b.session.Open(); err != nil {
		return err
	}

	b.registerDiscordCommands()

	b.logger.Info("discord bot started", "commands", len(b.commands))
	return nil
}

func (b *Bot) Stop() {
	if err := b.session.Close(); err != nil {
		b.logger.Warn("failed to close discord session", "error", err)
	}
}

func (b *Bot) registerDiscordCommands() {
	discordCmds := make([]*discordgo.ApplicationCommand, 0, len(b.commands))

	for _, cmd := range b.commands {
		discordCmds = append(discordCmds, commandToDiscord(cmd))
	}

	if guildID != "" {
		_, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, guildID, discordCmds)
		if err != nil {
			b.logger.Error("failed to register guild commands", "error", err)
		} else {
			b.logger.Info("registered guild commands", "count", len(discordCmds))
		}
	} else {
		_, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, "", discordCmds)
		if err != nil {
			b.logger.Error("failed to register global commands", "error", err)
		} else {
			b.logger.Info("registered global commands", "count", len(discordCmds))
		}
	}
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	cmdName := data.Name

	cmd, exists := b.commands[cmdName]
	if !exists {
		b.logger.Warn("unknown command received", "command", cmdName)
		return
	}

	ctx := &CommandContext{
		Session:     s,
		Interaction: i,
		Ctx:         context.Background(),
		Options:     parseOptions(i),
	}

	if err := cmd.Handle(ctx); err != nil {
		b.logger.Error("command handler failed", "command", cmdName, "error", err)
		if respondErr := ctx.Respond(formatCommandError(cmdName, err)); respondErr != nil {
			b.logger.Error("failed to send error response", "error", respondErr)
		}
	}
}
