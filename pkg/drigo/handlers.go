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

func (q *Bot) handlers() map[discordgo.InteractionType]map[string]pkg.Handler {
	return pkg.CommandHandlers{
		discordgo.InteractionApplicationCommand: {
			PostImageCommand: q.handlePostImage,
		},
		discordgo.InteractionApplicationCommandAutocomplete: {},
		discordgo.InteractionModalSubmit: {
			PostImageCommand: q.handlePostImage,
		},
	}
}

var lastImage utils.AttachmentImage

func (q *Bot) handlePostImage(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

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
