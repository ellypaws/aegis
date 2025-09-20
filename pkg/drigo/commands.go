package drigo

import (
	"github.com/bwmarrin/discordgo"
)

const (
	PostImageCommand = "post"
)

func (q *Bot) commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        PostImageCommand,
			Description: "Publish an image to subscribers",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				commandOption[fullImage],
				commandOption[thumbnailImage],
				commandOption[roleSelect],
			},
		},
	}
}

const (
	roleSelect     = "allowed_roles"
	thumbnailImage = "thumbnail"
	fullImage      = "full"
)

var commandOption = map[string]*discordgo.ApplicationCommandOption{
	fullImage: {
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        fullImage,
		Description: "Full Image",
		Required:    true,
	},
	thumbnailImage: {
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        thumbnailImage,
		Description: "Image",
		Required:    false,
	},
	roleSelect: {
		Type:        discordgo.ApplicationCommandOptionRole,
		Name:        roleSelect,
		Description: "Select a role to notify subscribers",
		Required:    false,
	},
}
