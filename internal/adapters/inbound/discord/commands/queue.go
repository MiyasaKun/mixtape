package commands

import (
	"fmt"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
)

// QueueCommand handles the /queue slash command: show the currently playing
// track and the tracks still waiting to be played.
type QueueCommand struct {
	musicPlayback *services.MusicPlaybackService
	queue         *services.QueueService
}

func NewQueueCommand(musicPlayback *services.MusicPlaybackService, queue *services.QueueService) *QueueCommand {
	return &QueueCommand{
		musicPlayback: musicPlayback,
		queue:         queue,
	}
}

func (c *QueueCommand) Name() string {
	return "queue"
}

func (c *QueueCommand) Description() string {
	return "Show the current queue"
}

func (c *QueueCommand) Options() []discord.CommandOption {
	return nil
}

func (c *QueueCommand) Handle(ctx *discord.CommandContext) error {
	guildID := ctx.GuildID()

	q, err := c.queue.GetQueue(ctx.Ctx, guildID)
	if err != nil {
		return ctx.Respond("❌ Failed to get queue: " + err.Error())
	}

	current, _ := c.musicPlayback.GetCurrentTrack(ctx.Ctx, guildID)
	if q == nil || len(q.Tracks) == 0 {
		if current != nil {
			return ctx.RespondEmbed("🎶 Queue", "▶️ **"+current.Title+"**\n\nNothing else queued")
		}
		return ctx.Respond("📭 Queue is empty")
	}

	description := ""
	if current != nil {
		description += "▶️ **" + current.Title + "**\n\n"
	}
	for i, track := range q.Tracks {
		description += fmt.Sprintf("%d. %s\n", i+1, track.Title)
	}

	return ctx.RespondEmbed("🎶 Queue", description)
}
