package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Name() string
	Description() string
	Options() []CommandOption
	Handle(ctx *CommandContext) error
}

type CommandOption struct {
	Type        discordgo.ApplicationCommandOptionType
	Name        string
	Description string
	Required    bool
	Choices     []*discordgo.ApplicationCommandOptionChoice
}

type CommandContext struct {
	Session     *discordgo.Session
	Interaction *discordgo.InteractionCreate
	Ctx         context.Context
	Options     map[string]interface{}
}

func (c *CommandContext) GetString(name string) string {
	if val, ok := c.Options[name]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (c *CommandContext) GetInt(name string) int {
	if val, ok := c.Options[name]; ok {
		if num, ok := val.(int); ok {
			return num
		}
	}
	return 0
}

func (c *CommandContext) GetBool(name string) bool {
	if val, ok := c.Options[name]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (c *CommandContext) Respond(content string) error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func (c *CommandContext) RespondEmbed(title, description string) error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       0x5865F2,
				},
			},
		},
	})
}

func (c *CommandContext) Defer() error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func (c *CommandContext) EditResponse(content string) error {
	_, err := c.Session.InteractionResponseEdit(c.Interaction.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	return err
}

func (c *CommandContext) GuildID() string {
	return c.Interaction.GuildID
}

func (c *CommandContext) UserID() string {
	return c.Interaction.Member.User.ID
}

func (c *CommandContext) ChannelID() string {
	return c.Interaction.ChannelID
}

func commandToDiscord(cmd Command) *discordgo.ApplicationCommand {
	opts := make([]*discordgo.ApplicationCommandOption, len(cmd.Options()))
	for i, opt := range cmd.Options() {
		opts[i] = &discordgo.ApplicationCommandOption{
			Type:        opt.Type,
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
			Choices:     opt.Choices,
		}
	}

	return &discordgo.ApplicationCommand{
		Name:        cmd.Name(),
		Description: cmd.Description(),
		Options:     opts,
	}
}

func parseOptions(i *discordgo.InteractionCreate) map[string]interface{} {
	options := make(map[string]interface{})
	data := i.ApplicationCommandData()

	for _, opt := range data.Options {
		switch opt.Type {
		case discordgo.ApplicationCommandOptionString:
			options[opt.Name] = opt.StringValue()
		case discordgo.ApplicationCommandOptionInteger:
			options[opt.Name] = int(opt.IntValue())
		case discordgo.ApplicationCommandOptionBoolean:
			options[opt.Name] = opt.BoolValue()
		case discordgo.ApplicationCommandOptionNumber:
			options[opt.Name] = opt.FloatValue()
		}
	}

	return options
}

func formatCommandError(cmd string, err error) string {
	return fmt.Sprintf("❌ Command `%s` failed: %v", cmd, err)
}
