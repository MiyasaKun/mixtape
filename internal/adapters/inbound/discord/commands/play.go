package commands

import (
	"context"
	"fmt"
	"time"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"

	dgo "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type PlayCommand struct {
	musicPlayback *services.MusicPlaybackService
}

func NewPlayCommand(musicPlayback *services.MusicPlaybackService) *PlayCommand {
	return &PlayCommand{
		musicPlayback: musicPlayback,
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

	//Generate a Connection for the Discord Guild
	conn := ctx.Client.VoiceManager.CreateConn(parsedGuildID)
	openCtx, cancel := context.WithTimeout(ctx.Ctx, 15*time.Second)
	defer cancel()

	// Use Connection to connect to Channel using DAVE
	err = conn.Open(openCtx, *voiceState.ChannelID, false, true)

	if err != nil {
		conn.Close(context.Background())
		return ctx.EditResponse(fmt.Sprintf("❌ Couldn't connect to voice: %v", err))
	}
	// TODO: Implement Play Function
	if err := c.musicPlayback.Play(ctx.Ctx, guildID, url); err != nil {
		return ctx.EditResponse("❌ Failed to play: " + err.Error())
	}

	return ctx.EditResponse("🎵 Now playing: " + url)
}
