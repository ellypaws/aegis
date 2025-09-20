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
	sendDM    = "send_dm"
)

func (q *Bot) components() map[string]pkg.Handler {
	return map[string]pkg.Handler{
		showImage: q.showImage,
		sendDM:    q.sendDM,
	}
}

var components = map[string]discordgo.MessageComponent{
	roleSelect: discordgo.SelectMenu{
		MenuType:    discordgo.RoleSelectMenu,
		CustomID:    roleSelect,
		Placeholder: "",
		MinValues:   utils.Pointer(1),
		MaxValues:   25,
		Disabled:    false,
	},
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
				Label:    "Send to DMS",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				Emoji: &discordgo.ComponentEmoji{
					Name: "📩",
				},
				CustomID: sendDM,
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

func (q *Bot) sendDM(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.EphemeralThink(s, i); err != nil {
		return nil
	}

	var userID string
	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	}
	if userID == "" {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "I couldn't figure out who to DM.", fmt.Errorf("no user id on interaction"))
	}

	if lastImage.Image == nil {
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    utils.Pointer("There's no image ready to send. Please make one first."),
			Components: &[]discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]},
		})
		return err
	}

	ch, err := s.UserChannelCreate(userID)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction,
			"I couldn't open your DMs. Please allow DMs from this server and try again.", err)
	}

	_, err = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: "📩 Here's your image — thanks for your support!",
		Files: []*discordgo.File{
			{
				Name:        "image.png",
				ContentType: "image/png",
				Reader:      lastImage.Image,
			},
		},
	})
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send you a DM.", err)
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    utils.Pointer("Sent! Check your DMs 📬"),
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]},
	})
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "DM sent, but I couldn't update the response.", err)
	}

	return nil
}
