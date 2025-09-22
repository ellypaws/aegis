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
				commandOption[postTitle],
				commandOption[postDescription],
				commandOption[roleSelect],
				commandOption[channelSelect],
			},
		},
	}
}

const (
	roleSelect      = "allowed_roles"
	channelSelect   = "allowed_channels"
	thumbnailImage  = "thumbnail"
	fullImage       = "full"
	postTitle       = "title"
	postDescription = "description"
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
		Required:    true,
	},
	roleSelect: {
		Type:        discordgo.ApplicationCommandOptionRole,
		Name:        roleSelect,
		Description: "Select role to notify subscribers. Do not select anything if you want to select multiple roles.",
		Required:    false,
	},
	channelSelect: {
		Type:        discordgo.ApplicationCommandOptionChannel,
		Name:        channelSelect,
		Description: "Select a channel to notify subscribers. Don't select if you want to include multiple channels.",
		Required:    false,
	},
	postTitle: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        postTitle,
		Description: "Optional title for the post",
		Required:    false,
	},
	postDescription: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        postDescription,
		Description: "Optional description for the post",
		Required:    false,
	},
}
