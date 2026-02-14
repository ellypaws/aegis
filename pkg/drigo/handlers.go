package drigo

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/log"
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

	// Parse selected roles and channels from modal components
	data := i.ModalSubmitData()
	var selectedRoles []string
	var selectedChannels []string
	var title, description string
	for _, c := range data.Components {
		if lbl, ok := c.(*discordgo.Label); ok {
			if sm, ok := lbl.Component.(*discordgo.SelectMenu); ok {
				switch sm.CustomID {
				case roleSelect:
					selectedRoles = append(selectedRoles, sm.Values...)
				case channelSelect:
					selectedChannels = append(selectedChannels, sm.Values...)
				}
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

	if pending.Author != nil {
		author := &types.User{}
		author.FromDiscord(pending.Author)
		post.Author = author
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
			BuildShowActionsRow(postKey),
		},
	}

	embed := &discordgo.MessageEmbed{
		Title:       post.Title,
		Type:        discordgo.EmbedTypeImage,
		Timestamp:   time.Now().Format(time.RFC3339),
		Description: post.Description,
	}
	if embed.Description == "" {
		embed.Description = "New image posted!"
	}

	if pending.Author != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    pending.Author.DisplayName(),
			IconURL: pending.Author.AvatarURL("64"),
			URL:     "https://discord.com/users/" + pending.Author.ID,
		}
	}
	if len(selectedChannels) == 0 {
		thumbR := bytes.NewReader(pending.Thumbnail)
		if err := handlers.EmbedImages(webhookEdit, embed, []io.Reader{thumbR}, nil, compositor.Compositor[*MemberExif](nil)); err != nil {
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

	var sentCount int
	for _, chID := range selectedChannels {
		if chID == "" {
			continue
		}
		perEmbed := *embed
		var perEdit discordgo.WebhookEdit
		if err := handlers.EmbedImages(&perEdit, &perEmbed, []io.Reader{bytes.NewReader(pending.Thumbnail)}, nil, compositor.Compositor[*MemberExif](nil)); err != nil {
			continue
		}
		msg, err := s.ChannelMessageSendComplex(chID, &discordgo.MessageSend{
			Content:    sb.String(),
			Files:      perEdit.Files,
			Embeds:     *perEdit.Embeds,
			Components: []discordgo.MessageComponent{BuildShowActionsRow(postKey)},
		})
		if err != nil {
			log.Error("Failed to post in selected channel", "channel", chID, "error", err)
			continue
		}
		log.Debug("Posted in selected channel", "channel", chID, "msg", msg.ID)
		q.mu.Lock()
		q.msgToPost[msg.ID] = postKey
		q.mu.Unlock()
		sentCount++
	}
	if sentCount == 0 {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to post in the selected channels.")
	}
	log.Debugf("Posted in %d/%d selected channels in %s", sentCount, len(selectedChannels), time.Since(now))

	if _, err := s.InteractionResponseEdit(i.Interaction, webhookEdit); err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Posted, but failed to update the response.", err)
	}

	q.mu.Lock()
	delete(q.pending, postKey)
	q.mu.Unlock()

	return nil
}

func (q *Bot) handlePostImage(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	now := time.Now()
	user := utils.GetUser(i.Interaction.Member)
	if user == nil {
		log.Errorf("Could not identify user for interaction [%s]", i.ID)
		return handlers.ErrorEphemeral(s, i.Interaction, "Could not identify you. Please try again.")
	}
	defer func() {
		log.Debugf("[%s] Handling /post command took %s", utils.GetDisplayName(user), time.Since(now))
	}()
	if user != nil {
		log.Info("Handling post image command", "user", user.DisplayName())
	}
	optionMap := utils.GetOpts(i.ApplicationCommandData())

	attachments, err := utils.GetAttachments(i)
	if err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error getting attachments.", err)
	}
	log.Debug("Attachments", "count", len(attachments), "took", time.Since(now))

	embed := &discordgo.MessageEmbed{
		Title:       "",
		Type:        discordgo.EmbedTypeImage,
		Timestamp:   time.Now().Format(time.RFC3339),
		Description: "New Image posted!",
	}
	if user != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    user.DisplayName(),
			IconURL: user.AvatarURL("64"),
			URL:     "https://discord.com/users/" + user.ID,
		}
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

		if !strings.Contains(thumbnailAttachment.Attachment.ContentType, "image/") {
			return handlers.ErrorEdit(s, i.Interaction, "The thumbnail must be an image.")
		}
	}

	if option, ok := optionMap[fullImage]; ok {
		image, ok := attachments[option.Value.(string)]
		if !ok {
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide a full image.")
		}
		fullAttachment = image
	}

	postKey := ksuid.New().String()
	if _, ok := optionMap[roleSelect]; !ok {
		log.Debugf("Responding with a modal in %s", time.Since(now))
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
					discordgo.Label{
						Label:       "Select Channel",
						Description: "You can select multiple channels to crosspost in.",
						Component:   components[channelSelect],
					},
					discordgo.Label{
						Label:       "Post title (optional)",
						Description: "Shown above the image.",
						Component: discordgo.TextInput{
							CustomID:    postTitle,
							Style:       discordgo.TextInputShort,
							Placeholder: "New Image posted!",
							Required:    false,
							MaxLength:   45,
							Value:       titleVal,
						},
					},
					discordgo.Label{
						Label:       "Post description (optional)",
						Description: "Shown under the title.",
						Component: discordgo.TextInput{
							CustomID:    postDescription,
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Add a short caption…",
							Required:    false,
							MaxLength:   2000,
							Value:       descVal,
						},
					},
				},
				Flags: discordgo.MessageFlagsIsComponentsV2,
			},
		})
		if err != nil {
			log.Error("Error responding with modal", "error", err)
			return handlers.ErrorEdit(s, i.Interaction, "Failed to open role selection modal", err)
		}

		thumbBytes := append([]byte(nil), thumbnailAttachment.Image.Bytes()...)
		fullBytes := append([]byte(nil), fullAttachment.Image.Bytes()...)

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

	role := optionMap[roleSelect].RoleValue(s, i.GuildID)

	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}
	log.Debug("First think response in ", time.Since(now))

	thumbBytes := append([]byte(nil), thumbnailAttachment.Image.Bytes()...)
	fullBytes := append([]byte(nil), fullAttachment.Image.Bytes()...)

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
	if user != nil {
		author := &types.User{}
		author.FromDiscord(user)
		post.Author = author
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
			BuildShowActionsRow(postKey),
		},
	}

	thumbR := bytes.NewReader(thumbBytes)
	if err := handlers.EmbedImages(webhookEdit, embed, []io.Reader{thumbR}, nil, compositor.Compositor[*MemberExif](nil)); err != nil {
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
