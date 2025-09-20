package drigo

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg"
	"drigo/pkg/compositor"
	"drigo/pkg/discord/handlers"
	"drigo/pkg/utils"
)

const (
	showImage = "show_image"
)

func (q *Bot) components() map[string]pkg.Handler {
	return map[string]pkg.Handler{
		showImage: q.showImage,
	}
}

var components = map[string]discordgo.MessageComponent{
	showImage: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Show me this image",
				Style:    discordgo.PrimaryButton,
				Disabled: false,
				Emoji: &discordgo.ComponentEmoji{
					Name: "⭐",
				},
				CustomID: showImage,
			},
			discordgo.Button{
				Label:    "Delete image",
				Style:    discordgo.DangerButton,
				Disabled: true,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🗑️",
				},
				CustomID: handlers.DeleteButton,
			},
		},
	},
}

func (q *Bot) showImage(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.EphemeralThink(s, i); err != nil {
		return nil
	}

	webhookEdit := &discordgo.WebhookEdit{
		Content: utils.Pointer("Thank you for your support! Here's your image:"),
		Components: &[]discordgo.MessageComponent{
			handlers.Components[handlers.DeleteButton],
		},
	}

	err := utils.EmbedImages(webhookEdit, nil, []io.Reader{lastImage.Image}, nil, compositor.Compositor())
	if err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}

	_, err = s.InteractionResponseEdit(i.Interaction, webhookEdit)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

	return nil
}
