package commands

import (
	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
)

type QueueCommand struct {
	queue *services.QueueService
}

func NewQueueCommand(queue *services.QueueService) *QueueCommand {
	return &QueueCommand{
		queue: queue,
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

	if q == nil || len(q.Tracks) == 0 {
		return ctx.Respond("📭 Queue is empty")
	}

	description := ""
	for i, track := range q.Tracks {
		marker := " "
		if i == q.CurrentIndex {
			marker = "▶️ "
		}
		description += marker + track.Title + "\n"
	}

	return ctx.RespondEmbed("🎶 Queue", description)
}
