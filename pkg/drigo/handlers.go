package drigo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/segmentio/ksuid"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg"
	"drigo/pkg/compositor"
	"drigo/pkg/discord/handlers"
	"drigo/pkg/utils"
)

func (q *Bot) handlers() map[discordgo.InteractionType]map[string]pkg.Handler {
	return pkg.CommandHandlers{
		discordgo.InteractionApplicationCommand: {
			PostImageCommand: q.handlePostImage,
		},
		discordgo.InteractionApplicationCommandAutocomplete: {},
		discordgo.InteractionModalSubmit: {
			PostImageCommand: q.handleModalSubmit,
		},
	}
}

var lastImage utils.AttachmentImage

func (q *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err := enc.Encode(i)
	if err != nil {
		log.Error("Failed to encode interaction", "error", err)
	}

	return nil
}

func (q *Bot) handlePostImage(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	optionMap := utils.GetOpts(i.ApplicationCommandData())

	attachments, err := utils.GetAttachments(i)
	if err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error getting attachments.", err)
	}

	var thumbnailAttachment utils.AttachmentImage
	if option, ok := optionMap[thumbnailImage]; ok {
		thumbnailAttachment, ok = attachments[option.Value.(string)]
		if !ok {
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide a thumbnail image.")
		}
	}

	if option, ok := optionMap[fullImage]; ok {
		image, ok := attachments[option.Value.(string)]
		if !ok {
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide a thumbnail image.")
		}
		lastImage = image
	}

	if option, ok := optionMap[roleSelect]; ok {
		role := option.RoleValue(s, i.GuildID)
		log.Info(role)
	} else {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: PostImageCommand + ksuid.New().String(),
				Title:    "Role selection is required",
				Components: []discordgo.MessageComponent{
					discordgo.TextDisplay{
						Content: "You must select at least one role that can view this image.",
					},
					discordgo.Label{
						Label:       "Select Role",
						Description: "You can select multiple roles.",
						Component:   components[roleSelect],
					},
				},
				Flags: discordgo.MessageFlagsIsComponentsV2,
			},
		})
		if err != nil {
			if err := handlers.ThinkResponse(s, i); err != nil {
				return err
			}
			return handlers.ErrorEdit(s, i.Interaction, "Failed to send response", err)
		}
	}

	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	webhookEdit := &discordgo.WebhookEdit{
		Content: nil,
		Components: &[]discordgo.MessageComponent{
			components[showImage],
		},
		Embeds:          nil,
		Files:           nil,
		Attachments:     nil,
		AllowedMentions: nil,
	}

	err = utils.EmbedImages(webhookEdit, nil, []io.Reader{thumbnailAttachment.Image}, nil, compositor.Compositor())
	if err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}

	_, err = s.InteractionResponseEdit(i.Interaction, webhookEdit)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

	return nil
}
