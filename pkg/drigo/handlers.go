package drigo

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/segmentio/ksuid"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg"
	"drigo/pkg/compositor"
	"drigo/pkg/discord/handlers"
	"drigo/pkg/types"
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
	// Extract our ksuid-based post key from the modal custom ID
	cid := i.ModalSubmitData().CustomID
	postKey := strings.TrimPrefix(cid, PostImageCommand)
	if postKey == "" {
		return handlers.ErrorEphemeral(s, i.Interaction, "Missing post key in modal submit")
	}

	q.mu.Lock()
	pending, ok := q.pending[postKey]
	q.mu.Unlock()
	if !ok || pending == nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "This post has expired or is invalid. Please try again.")
	}

	// Parse selected roles from modal components (RoleSelectMenu returns values = role IDs)
	data := i.ModalSubmitData()
	var selectedRoles []string
	for _, c := range data.Components {
		if lbl, ok := c.(*discordgo.Label); ok {
			if sm, ok := lbl.Component.(*discordgo.SelectMenu); ok {
				selectedRoles = append(selectedRoles, sm.Values...)
			}
		}
	}
	if len(selectedRoles) == 0 {
		return handlers.ErrorEphemeral(s, i.Interaction, "Please select at least one role.")
	}

	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	// Build post record
	now := time.Now().UTC()
	post := &types.Post{
		PostKey:   postKey,
		ChannelID: pending.ChannelID,
		GuildID:   pending.GuildID,
		Content:   "",
		Timestamp: now,
		IsPremium: true,
		Image: &types.Image{
			Thumbnail: append([]byte(nil), pending.Thumbnail...),
			Blobs: []types.ImageBlob{
				{Index: 0, Data: append([]byte(nil), pending.Full...)},
			},
		},
	}

	// Allowed roles association (store minimal info: RoleID)
	post.AllowedRoles = make([]types.Allowed, 0, len(selectedRoles))
	for _, rid := range selectedRoles {
		if rid == "" {
			continue
		}
		post.AllowedRoles = append(post.AllowedRoles, types.Allowed{RoleID: rid})
	}

	if err := q.db.CreatePost(post); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Failed to save post", err)
	}

	// Build response with thumbnail and buttons
	webhookEdit := &discordgo.WebhookEdit{
		Content: nil,
		Components: &[]discordgo.MessageComponent{
			components[showImage],
		},
	}

	// Use stable readers from bytes to avoid draining pending buffers
	thumbR := bytes.NewReader(pending.Thumbnail)
	if err := utils.EmbedImages(webhookEdit, nil, nil, []io.Reader{thumbR}, compositor.Compositor()); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, fmt.Errorf("error creating image embed: %w", err))
	}

	msg, err := s.InteractionResponseEdit(i.Interaction, webhookEdit)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

	// Map message ID to post key for button interactions
	q.mu.Lock()
	q.msgToPost[msg.ID] = postKey
	delete(q.pending, postKey)
	q.mu.Unlock()

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
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide a full image.")
		}
		lastImage = image // legacy fallback; DB will be the source of truth
	}

	// Prepare image bytes up-front to avoid draining streams twice
	thumbBytes := append([]byte(nil), thumbnailAttachment.Image.Bytes()...)
	var fullBytes []byte
	if lastImage.Image != nil {
		fullBytes = append([]byte(nil), lastImage.Image.Bytes()...)
	}

	// Generate a post key for this submission
	postKey := ksuid.New().String()

	// If no role provided in slash command, open modal to select multiple
	if _, ok := optionMap[roleSelect]; !ok {
		// Track pending post until modal submit
		q.mu.Lock()
		q.pending[postKey] = &PendingPost{
			PostKey:   postKey,
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    utils.GetUser(i.Interaction),
			Thumbnail: thumbBytes,
			Full:      fullBytes,
		}
		q.mu.Unlock()

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: PostImageCommand + postKey,
				Title:    "Role selection is required",
				Components: []discordgo.MessageComponent{
					discordgo.TextDisplay{Content: "You must select at least one role that can view this image."},
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
			return handlers.ErrorEdit(s, i.Interaction, "Failed to open role selection modal", err)
		}
		return nil
	}

	// Role provided directly in slash command (single role)
	role := optionMap[roleSelect].RoleValue(s, i.GuildID)

	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	// Persist post immediately
	now := time.Now().UTC()
	post := &types.Post{
		PostKey:   postKey,
		ChannelID: i.ChannelID,
		GuildID:   i.GuildID,
		Content:   "",
		Timestamp: now,
		IsPremium: true,
		Image: &types.Image{
			Thumbnail: append([]byte(nil), thumbBytes...),
			Blobs:     []types.ImageBlob{{Index: 0, Data: append([]byte(nil), fullBytes...)}},
		},
	}
	if role != nil {
		post.AllowedRoles = []types.Allowed{{
			RoleID:       role.ID,
			Name:         role.Name,
			Managed:      role.Managed,
			Mentionable:  role.Mentionable,
			Hoist:        role.Hoist,
			Color:        role.Color,
			Position:     role.Position,
			Permissions:  role.Permissions,
			Icon:         role.Icon,
			UnicodeEmoji: role.UnicodeEmoji,
			Flags:        int(role.Flags),
		}}
	}

	if err := q.db.CreatePost(post); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Failed to save post", err)
	}

	// Build public message with thumbnail and buttons
	webhookEdit := &discordgo.WebhookEdit{
		Content: nil,
		Components: &[]discordgo.MessageComponent{
			components[showImage],
		},
	}
	// Use stable readers from bytes
	thumbR := bytes.NewReader(thumbBytes)
	if err := utils.EmbedImages(webhookEdit, nil, nil, []io.Reader{thumbR}, compositor.Compositor()); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, fmt.Errorf("error creating image embed: %w", err))
	}

	msg, err := s.InteractionResponseEdit(i.Interaction, webhookEdit)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

	q.mu.Lock()
	q.msgToPost[msg.ID] = postKey
	q.mu.Unlock()

	return nil
}
