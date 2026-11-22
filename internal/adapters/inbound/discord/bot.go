package discord

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"time"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/logging"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	dgo "github.com/disgoorg/disgo/discord"
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
	metrics            ports.Metrics
	commands           map[string]Command
	musicPlayback      *services.MusicPlaybackService
	queue              *services.QueueService
	guildConfig        *services.GuildConfigService
	spotifyIntegration *services.SpotifyIntegrationService
	selections         *SelectionStore
}

func NewBot(
	token string,
	logger *logging.Logger,
	metrics ports.Metrics,
	musicPlayback *services.MusicPlaybackService,
	queue *services.QueueService,
	guildConfig *services.GuildConfigService,
	spotifyIntegration *services.SpotifyIntegrationService,
) *Bot {
	if metrics == nil {
		metrics = ports.NewNoopMetrics()
	}
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
		metrics:            metrics,
		commands:           make(map[string]Command),
		musicPlayback:      musicPlayback,
		queue:              queue,
		guildConfig:        guildConfig,
		spotifyIntegration: spotifyIntegration,
		selections:         NewSelectionStore(SelectionTTL + 5*time.Second),
	}

	client.AddEventListeners(
		bot.NewListenerFunc(b.handleInteraction),
		bot.NewListenerFunc(b.handleComponent),
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

// Selections exposes the pending Spotify selection store for command wiring.
func (b *Bot) Selections() *SelectionStore {
	return b.selections
}

// ActiveVoiceGuilds lists guilds with an open voice connection. Connections
// that were created but never opened (no channel yet) are not playable and
// are excluded so the web UI only offers real playback targets.
func (b *Bot) ActiveVoiceGuilds() []ports.VoiceGuild {
	return ActiveVoiceGuilds(b.client.VoiceManager.Conns(), b.guildNameByID)
}

// ActiveVoiceGuilds maps open voice connections to their guilds. guildName
// resolves a display name and reports false when the guild is unknown.
func ActiveVoiceGuilds(conns iter.Seq[voice.Conn], guildName func(snowflake.ID) (string, bool)) []ports.VoiceGuild {
	guilds := make([]ports.VoiceGuild, 0)
	for conn := range conns {
		if conn.ChannelID() == nil {
			continue
		}
		guildID := conn.GuildID()
		name := ""
		if resolved, ok := guildName(guildID); ok {
			name = resolved
		}
		guilds = append(guilds, ports.VoiceGuild{ID: guildID.String(), Name: name})
	}
	sort.Slice(guilds, func(i, j int) bool {
		if guilds[i].Name == guilds[j].Name {
			return guilds[i].ID < guilds[j].ID
		}
		return guilds[i].Name < guilds[j].Name
	})
	return guilds
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

	log := b.logger.With(
		"command", cmdName,
		"guild_id", guildIDString(e.GuildID()),
		"guild_name", b.guildName(e.GuildID()),
		"user_id", e.User().ID.String(),
		"user_name", e.User().EffectiveName(),
	)

	cmd, exists := b.commands[cmdName]
	if !exists {
		log.Warn("unknown command received")
		b.metrics.CommandHandled(cmdName, "unknown")
		return
	}

	ctx := &CommandContext{
		Client:      b.client,
		Interaction: e,
		Ctx:         context.Background(),
		Options:     parseOptions(e),
	}

	log.Info("command received", "options", ctx.Options)

	if err := cmd.Handle(ctx); err != nil {
		log.Error("command handler failed", "error", err)
		b.metrics.CommandHandled(cmdName, "error")
		if respondErr := ctx.Respond(formatCommandError(cmdName, err)); respondErr != nil {
			log.Error("failed to send error response", "respond_error", respondErr)
		}
		return
	}
	b.metrics.CommandHandled(cmdName, "ok")
	log.Debug("command completed")
}

// guildIDString renders an optional guild ID for logging attributes.
func guildIDString(guildID *snowflake.ID) string {
	if guildID == nil {
		return ""
	}
	return guildID.String()
}

// guildNameByID resolves a guild display name from the cache, reporting
// whether the guild is known.
func (b *Bot) guildNameByID(guildID snowflake.ID) (string, bool) {
	guild, ok := b.client.Caches.Guild(guildID)
	if !ok {
		return "", false
	}
	return guild.Name, true
}

// guildName resolves the guild's display name from the cache. It returns an
// empty string for DM interactions or cache misses so logging never adds a
// REST round-trip.
func (b *Bot) guildName(guildID *snowflake.ID) string {
	if guildID == nil {
		return ""
	}
	guild, ok := b.client.Caches.Guild(*guildID)
	if !ok {
		return ""
	}
	return guild.Name
}

// handleComponent routes component interactions. Everything goes to the
// Spotify pick flow; unknown custom IDs are ignored. The nowplaying card is
// purely visual and has no components to route.
func (b *Bot) handleComponent(e *events.ComponentInteractionCreate) {
	guildID, userID, requestID, index, ok := ParseSpotifyPickCustomID(e.Data.CustomID())
	if !ok {
		return
	}

	// Discord may omit GuildID in DMs; the flow is guild-only.
	if e.GuildID() == nil || e.GuildID().String() != guildID {
		b.ephemeralError(e, "❌ This selection is no longer valid")
		return
	}
	if e.User().ID.String() != userID {
		b.ephemeralError(e, "❌ Only the user who ran `/play` can pick a track")
		return
	}

	candidates, coverURL, ok := b.selections.Take(requestID)
	if !ok {
		b.ephemeralError(e, "❌ This selection expired — use `/play` again")
		return
	}

	if index < 0 || index >= len(candidates) {
		b.ephemeralError(e, "❌ Invalid choice — use `/play` again")
		return
	}
	chosen := candidates[index]

	content := "⏳ Loading song..."
	emptyComponents := []dgo.LayoutComponent{}
	if err := e.UpdateMessage(discord.MessageUpdate{Content: &content, Components: &emptyComponents}); err != nil {
		b.logger.Error("failed to update spotify pick message", "error", err)
	}

	// Playback involves yt-dlp and ffmpeg; do not block the event loop.
	go func() {
		var content string
		result, err := b.musicPlayback.PlayWithCover(context.Background(), guildID, chosen.URL, coverURL)
		if err != nil {
			b.logger.Error("failed to play selected spotify track", "error", err)
			content = "❌ Failed to play: " + err.Error()
		} else {
			content = fmt.Sprintf("🎵 Now playing: **%s**", result.Track.Title)
		}
		if _, err := b.client.Rest.UpdateInteractionResponse(b.client.ApplicationID, e.Token(), discord.MessageUpdate{Content: &content}); err != nil {
			b.logger.Error("failed to report playback result", "error", err)
		}
	}()
}

// ephemeralError answers a component interaction with an ephemeral notice so
// other users do not see it.
func (b *Bot) ephemeralError(e *events.ComponentInteractionCreate, content string) {
	err := e.CreateMessage(discord.MessageCreate{
		Content: content,
		Flags:   discord.MessageFlagEphemeral,
	})
	if err != nil {
		b.logger.Error("failed to send ephemeral error", "error", err)
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
