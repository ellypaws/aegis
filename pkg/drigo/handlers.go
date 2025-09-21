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
	var title, description string
	for _, c := range data.Components {
		if lbl, ok := c.(*discordgo.Label); ok {
			if sm, ok := lbl.Component.(*discordgo.SelectMenu); ok {
				selectedRoles = append(selectedRoles, sm.Values...)
				continue
			}
			if ti, ok := lbl.Component.(*discordgo.TextInput); ok {
				switch ti.CustomID {
				case postTitle:
					title = ti.Value
				case postDescription:
					description = ti.Value
				}
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
		PostKey:     postKey,
		ChannelID:   pending.ChannelID,
		GuildID:     pending.GuildID,
		Title:       pending.Title,
		Description: pending.Description,
		Timestamp:   now,
		IsPremium:   true,
		Image: &types.Image{
			Thumbnail: append([]byte(nil), pending.Thumbnail...),
			Blobs: []types.ImageBlob{
				{Index: 0, Data: append([]byte(nil), pending.Full...)},
			},
		},
	}

	// If modal provided text values, they override pending defaults
	if title != "" {
		post.Title = title
	}
	if description != "" {
		post.Description = description
	}

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

	var sb strings.Builder
	sb.WriteString("> Allowed roles:\n")
	for _, r := range post.AllowedRoles {
		sb.WriteString("> <@&")
		sb.WriteString(r.RoleID)
		sb.WriteString(">\n")
	}

	webhookEdit := &discordgo.WebhookEdit{
		Content: utils.Pointer(sb.String()),
		Components: &[]discordgo.MessageComponent{
			buildShowActionsRow(postKey),
		},
	}

	// Always include an embed that carries title/description; fall back description for UX only
	embed := &discordgo.MessageEmbed{
		Title:     post.Title,
		Type:      discordgo.EmbedTypeImage,
		Timestamp: time.Now().Format(time.RFC3339),
		Description: func() string {
			if post.Description != "" {
				return post.Description
			}
			return "New Image posted!"
		}(),
	}
	thumbR := bytes.NewReader(pending.Thumbnail)
	if err := utils.EmbedImages(webhookEdit, embed, []io.Reader{thumbR}, nil, compositor.Compositor()); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, fmt.Errorf("error creating image embed: %w", err))
	}

	msg, err := s.InteractionResponseEdit(i.Interaction, webhookEdit)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

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

	embed := &discordgo.MessageEmbed{
		Title:       "",
		Type:        discordgo.EmbedTypeImage,
		Timestamp:   time.Now().Format(time.RFC3339),
		Description: "New Image posted!",
	}
	var titleVal, descVal string
	if titleOpt, ok := optionMap[postTitle]; ok && titleOpt.Value != nil {
		if s, ok := titleOpt.Value.(string); ok {
			titleVal = s
			embed.Title = s
		}
	}
	if descOpt, ok := optionMap[postDescription]; ok && descOpt.Value != nil {
		if s, ok := descOpt.Value.(string); ok {
			descVal = s
			embed.Description = s
		}
	}

	var thumbnailAttachment utils.AttachmentImage
	var fullAttachment utils.AttachmentImage
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
		fullAttachment = image
	}

	thumbBytes := append([]byte(nil), thumbnailAttachment.Image.Bytes()...)
	fullBytes := append([]byte(nil), fullAttachment.Image.Bytes()...)

	postKey := ksuid.New().String()
	if _, ok := optionMap[roleSelect]; !ok {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: PostImageCommand + postKey,
				Title:    "Role selection is required",
				Components: []discordgo.MessageComponent{
					discordgo.TextDisplay{Content: "You must select at least one role that can view this image."},
					// Optional title input
					discordgo.Label{
						Label:       "Post title (optional)",
						Description: "Shown above the image.",
						Component: discordgo.TextInput{
							CustomID:    postTitle,
							Label:       "Title",
							Style:       discordgo.TextInputShort,
							Placeholder: "New Image posted!",
							Required:    false,
							MaxLength:   256,
							Value:       titleVal,
						},
					},
					// Optional description input
					discordgo.Label{
						Label:       "Post description (optional)",
						Description: "Shown under the title.",
						Component: discordgo.TextInput{
							CustomID:    postDescription,
							Label:       "Description",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Add a short caption…",
							Required:    false,
							MaxLength:   2000,
							Value:       descVal,
						},
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
			return handlers.ErrorEdit(s, i.Interaction, "Failed to open role selection modal", err)
		}

		q.mu.Lock()
		q.pending[postKey] = &PendingPost{
			PostKey:     postKey,
			GuildID:     i.GuildID,
			ChannelID:   i.ChannelID,
			Author:      utils.GetUser(i.Interaction),
			Thumbnail:   thumbBytes,
			Full:        fullBytes,
			Title:       titleVal,
			Description: descVal,
		}
		q.mu.Unlock()

		return nil
	}

	// Role provided directly in slash command (single role)
	role := optionMap[roleSelect].RoleValue(s, i.GuildID)

	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	now := time.Now().UTC()
	post := &types.Post{
		PostKey:     postKey,
		ChannelID:   i.ChannelID,
		GuildID:     i.GuildID,
		Title:       titleVal,
		Description: descVal,
		Timestamp:   now,
		IsPremium:   true,
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

	webhookEdit := &discordgo.WebhookEdit{
		Content: nil,
		Components: &[]discordgo.MessageComponent{
			buildShowActionsRow(postKey),
		},
	}

	thumbR := bytes.NewReader(thumbBytes)
	if err := utils.EmbedImages(webhookEdit, embed, []io.Reader{thumbR}, nil, compositor.Compositor()); err != nil {
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
