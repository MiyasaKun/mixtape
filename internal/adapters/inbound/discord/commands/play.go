package commands

import (
	"context"
	"fmt"
	"time"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/logging"

	dgo "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// PlayCommand handles the /play slash command: join the caller's voice
// channel (if not already connected) and queue up the given track or search
// query for playback. Spotify links are answered with a select menu of
// YouTube candidates for the caller to pick from.
type PlayCommand struct {
	musicPlayback *services.MusicPlaybackService
	resolver      *services.TrackResolverService
	selections    *discord.SelectionStore
	logger        *logging.Logger
}

func NewPlayCommand(
	musicPlayback *services.MusicPlaybackService,
	resolver *services.TrackResolverService,
	selections *discord.SelectionStore,
	logger *logging.Logger,
) *PlayCommand {
	return &PlayCommand{
		musicPlayback: musicPlayback,
		resolver:      resolver,
		selections:    selections,
		logger:        logger,
	}
}

func (c *PlayCommand) Name() string {
	return "play"
}

func (c *PlayCommand) Description() string {
	return "Play a track from YouTube, Spotify, or SoundCloud"
}

func (c *PlayCommand) Options() []discord.CommandOption {
	return []discord.CommandOption{
		{
			Type:        dgo.ApplicationCommandOptionTypeString,
			Name:        "url",
			Description: "Track URL or search query",
			Required:    true,
		},
	}
}

func (c *PlayCommand) Handle(ctx *discord.CommandContext) error {
	// Defer the reply so we can take our time resolving and starting the
	// track, then edit the response with the final result.
	err := ctx.Defer()

	if err != nil {
		return err
	}

	url := ctx.GetString("url")

	guildID := ctx.GuildID()
	if guildID == "" {
		return ctx.EditResponse("❌ `/play` can only be used in a server")
	}
	userId := ctx.Interaction.User().ID

	parsedGuildID, err := snowflake.Parse(guildID)
	if err != nil {
		return ctx.EditResponse("❌ Invalid server ID: " + err.Error())
	}

	// Get the Voice State of the Requesting User
	voiceState, ok := ctx.Client.Caches.VoiceState(parsedGuildID, userId)
	if !ok || voiceState.ChannelID == nil {
		return ctx.EditResponse("❌ Join a voice channel before using `/play`")
	}

	//Generate a Connection for the Discord Guild. CreateConn returns the
	// existing connection if the bot is already connected to this guild.
	conn := ctx.Client.VoiceManager.CreateConn(parsedGuildID)

	// Only open the connection if the bot is not already in the target
	// channel. Re-opening an already-open connection would wait forever for
	// a new session description that never arrives and time out.
	if ch := conn.ChannelID(); ch == nil || *ch != *voiceState.ChannelID {
		openCtx, cancel := context.WithTimeout(ctx.Ctx, 15*time.Second)
		defer cancel()

		// Use Connection to connect to Channel using DAVE
		err = conn.Open(openCtx, *voiceState.ChannelID, false, false)

		if err != nil {
			conn.Close(context.Background())
			return ctx.EditResponse(fmt.Sprintf("❌ Couldn't connect to voice: %v", err))
		}
	}

	// Register the connection with the playback service.
	c.musicPlayback.SetVoiceConn(guildID, discord.NewVoiceConnAdapter(conn))

	// Spotify links first offer a choice between YouTube candidates.
	if services.IsSpotifyURL(url) {
		return c.handleSpotifyLink(ctx, guildID, url)
	}

	result, err := c.musicPlayback.Play(ctx.Ctx, guildID, url)
	if err != nil {
		c.musicPlayback.Disconnect(context.Background(), guildID)
		return ctx.EditResponse("❌ Failed to play: " + err.Error())
	}

	if result.Started {
		return ctx.EditResponse(fmt.Sprintf("🎵 Now playing: **%s**", result.Track.Title))
	}
	return ctx.EditResponse(fmt.Sprintf("➕ Added to queue (position %d): **%s**", result.Position, result.Track.Title))
}

// handleSpotifyLink resolves a Spotify track link into YouTube candidates,
// posts a select menu with them, and edits the message to a timeout notice if
// the caller does not pick a track in time.
func (c *PlayCommand) handleSpotifyLink(ctx *discord.CommandContext, guildID, spotifyURL string) error {
	candidates, coverURL, err := c.resolver.SpotifyYouTubeOptions(ctx.Ctx, spotifyURL)
	if err != nil {
		return ctx.EditResponse("❌ " + err.Error())
	}
	if len(candidates) == 0 {
		return ctx.EditResponse("❌ No YouTube results found for this Spotify track")
	}

	requestID := c.selections.Put(ctx.UserID(), guildID, coverURL, candidates)
	buttons := discord.SpotifySelectMenuComponent(guildID, ctx.UserID(), requestID, candidates)
	content := "🔎 Found this track on Spotify. Which YouTube version should I play?\n\n" +
		discord.SpotifyPickList(candidates)
	components := &buttons
	_, err = ctx.Client.Rest.UpdateInteractionResponse(ctx.Client.ApplicationID, ctx.Interaction.Token(), dgo.MessageUpdate{
		Content:    &content,
		Components: components,
	})
	if err != nil {
		return err
	}

	// Give the caller the selection window to pick; afterwards edit the
	// message to a timeout notice. Take misses when the user already picked,
	// in which case the menu interaction handled the message.
	token := ctx.Interaction.Token()
	appID := ctx.Client.ApplicationID
	time.AfterFunc(discord.SelectionTTL, func() {
		if _, _, stillPending := c.selections.Take(requestID); stillPending {
			// Nobody picked within the window.
			content := "⏱ Selection timed out — no track was played. Use `/play` again to retry."
			emptyComponents := []dgo.LayoutComponent{}
			if _, err := ctx.Client.Rest.UpdateInteractionResponse(appID, token, dgo.MessageUpdate{Content: &content, Components: &emptyComponents}); err != nil {
				c.logger.Error("failed to edit spotify selection timeout", "error", err)
			}
		}
	})
	return nil
}
