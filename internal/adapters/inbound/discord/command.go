package discord

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

type Command interface {
	Name() string
	Description() string
	Options() []CommandOption
	Handle(ctx *CommandContext) error
}

type CommandOption struct {
	Type        discord.ApplicationCommandOptionType
	Name        string
	Description string
	Required    bool
	Choices     []discord.ApplicationCommandOptionChoice
}

type CommandContext struct {
	Client      *bot.Client
	Interaction *events.ApplicationCommandInteractionCreate
	Ctx         context.Context
	Options     map[string]interface{}
	deferred    bool
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
	if c.deferred {
		return c.EditResponse(content)
	}

	return c.Interaction.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageCreate{
		Content: content,
	})
}

func (c *CommandContext) RespondEmbed(title, description string) error {
	return c.Interaction.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				Title:       title,
				Description: description,
				Color:       0x5865F2,
			},
		},
	})
}

func (c *CommandContext) Defer() error {
	err := c.Interaction.Respond(discord.InteractionResponseTypeDeferredCreateMessage, nil)
	if err == nil {
		c.deferred = true
	}
	return err
}

func (c *CommandContext) EditResponse(content string) error {
	_, err := c.Client.Rest.UpdateInteractionResponse(c.Client.ApplicationID, c.Interaction.Token(), discord.MessageUpdate{
		Content: &content,
	})
	return err
}

func (c *CommandContext) GuildID() string {
	if guildID := c.Interaction.GuildID(); guildID != nil {
		return guildID.String()
	}
	return ""
}

func (c *CommandContext) UserID() string {
	return c.Interaction.User().ID.String()
}

func (c *CommandContext) ChannelID() string {
	return c.Interaction.Channel().ID().String()
}

func commandToDiscord(cmd Command) discord.ApplicationCommandCreate {
	opts := make([]discord.ApplicationCommandOption, len(cmd.Options()))
	for i, opt := range cmd.Options() {
		opts[i] = optionToDiscord(opt)
	}

	return discord.SlashCommandCreate{
		Name:        cmd.Name(),
		Description: cmd.Description(),
		Options:     opts,
	}
}

func optionToDiscord(opt CommandOption) discord.ApplicationCommandOption {
	switch opt.Type {
	case discord.ApplicationCommandOptionTypeString:
		return discord.ApplicationCommandOptionString{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
			Choices:     choicesToString(opt.Choices),
		}
	case discord.ApplicationCommandOptionTypeInt:
		return discord.ApplicationCommandOptionInt{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
			Choices:     choicesToInt(opt.Choices),
		}
	case discord.ApplicationCommandOptionTypeBool:
		return discord.ApplicationCommandOptionBool{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	case discord.ApplicationCommandOptionTypeFloat:
		return discord.ApplicationCommandOptionFloat{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
			Choices:     choicesToFloat(opt.Choices),
		}
	case discord.ApplicationCommandOptionTypeChannel:
		return discord.ApplicationCommandOptionChannel{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	case discord.ApplicationCommandOptionTypeUser:
		return discord.ApplicationCommandOptionUser{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	case discord.ApplicationCommandOptionTypeRole:
		return discord.ApplicationCommandOptionRole{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	case discord.ApplicationCommandOptionTypeMentionable:
		return discord.ApplicationCommandOptionMentionable{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	case discord.ApplicationCommandOptionTypeAttachment:
		return discord.ApplicationCommandOptionAttachment{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	default:
		return discord.ApplicationCommandOptionString{
			Name:        opt.Name,
			Description: opt.Description,
			Required:    opt.Required,
		}
	}
}

func choicesToString(choices []discord.ApplicationCommandOptionChoice) []discord.ApplicationCommandOptionChoiceString {
	if len(choices) == 0 {
		return nil
	}
	result := make([]discord.ApplicationCommandOptionChoiceString, 0, len(choices))
	for _, choice := range choices {
		if c, ok := choice.(discord.ApplicationCommandOptionChoiceString); ok {
			result = append(result, c)
		}
	}
	return result
}

func choicesToInt(choices []discord.ApplicationCommandOptionChoice) []discord.ApplicationCommandOptionChoiceInt {
	if len(choices) == 0 {
		return nil
	}
	result := make([]discord.ApplicationCommandOptionChoiceInt, 0, len(choices))
	for _, choice := range choices {
		if c, ok := choice.(discord.ApplicationCommandOptionChoiceInt); ok {
			result = append(result, c)
		}
	}
	return result
}

func choicesToFloat(choices []discord.ApplicationCommandOptionChoice) []discord.ApplicationCommandOptionChoiceFloat {
	if len(choices) == 0 {
		return nil
	}
	result := make([]discord.ApplicationCommandOptionChoiceFloat, 0, len(choices))
	for _, choice := range choices {
		if c, ok := choice.(discord.ApplicationCommandOptionChoiceFloat); ok {
			result = append(result, c)
		}
	}
	return result
}

func parseOptions(e *events.ApplicationCommandInteractionCreate) map[string]interface{} {
	options := make(map[string]interface{})
	data := e.SlashCommandInteractionData()

	for _, opt := range data.All() {
		switch opt.Type {
		case discord.ApplicationCommandOptionTypeString:
			options[opt.Name] = opt.String()
		case discord.ApplicationCommandOptionTypeInt:
			options[opt.Name] = opt.Int()
		case discord.ApplicationCommandOptionTypeBool:
			options[opt.Name] = opt.Bool()
		case discord.ApplicationCommandOptionTypeFloat:
			options[opt.Name] = opt.Float()
		case discord.ApplicationCommandOptionTypeChannel, discord.ApplicationCommandOptionTypeUser, discord.ApplicationCommandOptionTypeRole, discord.ApplicationCommandOptionTypeMentionable:
			options[opt.Name] = opt.Snowflake().String()
		}
	}

	return options
}

func formatCommandError(cmd string, err error) string {
	return fmt.Sprintf("❌ Command `%s` failed: %v", cmd, err)
}
