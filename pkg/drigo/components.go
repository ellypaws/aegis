package drigo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"drigo/pkg/exif"
	"drigo/pkg/types"
	"drigo/pkg/units"

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
		MinValues:   new(0),
		MaxValues:   25,
		Disabled:    false,
		Required:    new(false),
	},
	channelSelect: discordgo.SelectMenu{
		MenuType: discordgo.ChannelSelectMenu,
		CustomID: channelSelect,
		ChannelTypes: []discordgo.ChannelType{
			discordgo.ChannelTypeGuildText,
			discordgo.ChannelTypeGuildVoice,
			discordgo.ChannelTypeGuildForum,
			discordgo.ChannelTypeGuildMedia,
		},
		Placeholder: "",
		MinValues:   new(0),
		MaxValues:   25,
		Disabled:    false,
		Required:    new(false),
	},
	thumbnailModal: discordgo.FileUpload{
		CustomID: thumbnailModal,
		Required: new(false),
	},
}

// BuildShowActionsRow returns the two-button row (Show image + Send to DMs) with embedded postKey.
func BuildShowActionsRow(postKey string) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Show me this image",
				Style:    discordgo.PrimaryButton,
				Disabled: false,
				Emoji:    &discordgo.ComponentEmoji{Name: "⭐"},
				CustomID: showImage + ":" + postKey,
			},
			discordgo.Button{
				Label:    "Send to DMs",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				Emoji:    &discordgo.ComponentEmoji{Name: "📩"},
				CustomID: sendDM + ":" + postKey,
			},
		},
	}
}

// buildThumbnailModalLabel returns an optional Label+FileUpload component for modal prompts
// when no thumbnail was provided with the original command. Callers may conditionally append
// the returned component to a modal's component list.
func buildThumbnailModalLabel() discordgo.Label {
	return discordgo.Label{
		Label:       "Upload Thumbnail (optional)",
		Description: "Provide a preview image for the post. One will be auto-generated if skipped.",
		Component:   components[thumbnailModal],
	}
}

// buildDMOnlyRow returns a single-button row for ephemeral follow-ups.
func buildDMOnlyRow(postKey string) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Send to DMs",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				Emoji:    &discordgo.ComponentEmoji{Name: "📩"},
				CustomID: sendDM + ":" + postKey,
			},
		},
	}
}

func (q *Bot) showImage(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.EphemeralThink(s, i); err != nil {
		return nil
	}

	// Prefer postKey encoded in the component CustomID; fallback to in-memory map for legacy messages.
	var postKey string
	cid := i.MessageComponentData().CustomID
	if strings.HasPrefix(cid, showImage+":") {
		postKey = strings.TrimPrefix(cid, showImage+":")
	} else if strings.HasPrefix(cid, sendDM+":") {
		postKey = strings.TrimPrefix(cid, sendDM+":")
	}
	if postKey == "" {
		q.mu.Lock()
		postKey = q.msgToPost[i.Message.ID]
		q.mu.Unlock()
	}
	if postKey == "" {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "I couldn't link this button to a post.")
	}

	p, err := q.db.ReadPostByExternalID(postKey)
	if err != nil || p == nil || len(p.Images) == 0 {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Couldn't load the image for this post.", err)
	}

	member, allowed := q.isAllowedToView(s, i, p)
	if !allowed {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "You don't have access to view this image.")
	}

	webhookEdit := &discordgo.WebhookEdit{
		Content:    utils.Pointer("Thank you for your support! Here's your image:"),
		Components: &[]discordgo.MessageComponent{buildDMOnlyRow(postKey)},
	}

	embed := &discordgo.MessageEmbed{
		Title:       p.Title,
		Type:        discordgo.EmbedTypeImage,
		Timestamp:   p.Timestamp.Format(time.RFC3339),
		Description: p.Description,
	}
	if embed.Description == "" {
		embed.Description = "New image posted!"
	}

	// Show the original author on the embed if available
	if p.Author != nil {
		if du := p.Author.ToDiscord(); du != nil {
			embed.Author = &discordgo.MessageEmbedAuthor{
				Name:    du.DisplayName(),
				IconURL: du.AvatarURL("64"),
				URL:     "https://discord.com/users/" + du.ID,
			}
		}
	}

	err = q.prepareEmbed(member, p, webhookEdit, embed)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to prepare embed", err)
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, webhookEdit); err != nil {
		var restErr *discordgo.RESTError
		if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeRequestEntityTooLarge {
			url, fbEmbed, err := q.fallbackToLink(i, p, 0)
			if err != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction,
					"The image was too large to send and no fallback storage is configured.", err)
			}

			if _, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.Pointer(fmt.Sprintf("📎 This image was too large to send directly. Here's a link: %s", url)),
				Embeds:  &[]*discordgo.MessageEmbed{fbEmbed},
				Components: &[]discordgo.MessageComponent{
					buildDMOnlyRow(postKey),
				},
			}); editErr != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Uploaded, but couldn't update the response.", editErr)
			}
			return nil
		}
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

	return nil
}

func (q *Bot) prepareEmbed(member *discordgo.Member, p *types.Post, webhookEdit *discordgo.WebhookEdit, embed *discordgo.MessageEmbed) error {
	if len(p.Images) == 0 {
		return fmt.Errorf("no images in post")
	}
	// Only embed the first image for public/embed view
	firstImage := p.Images[0]

	if len(p.Images) > 1 {
		if embed.Footer == nil {
			embed.Footer = &discordgo.MessageEmbedFooter{}
		}
		embed.Footer.Text = fmt.Sprintf(" +%d more image(s) available on the server", len(p.Images)-1)
	}

	if firstImage.HasVideo() {
		if err := handlers.EmbedImages(webhookEdit, embed, firstImage.Readers(), nil, compositor.Passthrough); err != nil {
			return fmt.Errorf("error creating image embed: %w", err)
		}
		return nil
	}

	memberExif := types.ToMemberExif(member)
	encoder := exif.NewEncoder(memberExif)
	var images []io.Reader
	for _, blob := range firstImage.Blobs {
		imgBlob, err := q.db.GetImageBlob(blob.ID)
		if err != nil {
			return fmt.Errorf("error getting image blob: %w", err)
		}

		if imgBlob.ContentType == "image/png" {
			var buffer bytes.Buffer
			err := encoder.EncodeReader(&buffer, bytes.NewReader(imgBlob.Data))
			if err != nil {
				return fmt.Errorf("error encoding: %w", err)
			}
			images = append(images, &buffer)
		} else {
			images = append(images, bytes.NewReader(imgBlob.Data))
		}
	}

	if err := handlers.EmbedImages(webhookEdit, embed, images, nil, compositor.Compositor(memberExif)); err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}
	return nil
}

func (q *Bot) isAllowedToView(s *discordgo.Session, i *discordgo.InteractionCreate, p *types.Post) (*discordgo.Member, bool) {
	member := i.Member
	if member == nil {
		if i.GuildID != "" {
			user := utils.GetUser(i.Member, i.User)
			if user != nil {
				if m, err := s.State.Member(i.GuildID, user.ID); err == nil {
					member = m
				} else if m, err := s.GuildMember(i.GuildID, user.ID); err == nil {
					member = m
				}
			}
		}
	}

	if len(p.AllowedRoles) == 0 {
		return member, true
	}

	if member == nil {
		log.Warn("no member info available to check roles")
		return nil, false
	}

	roleSet := map[string]struct{}{}
	for _, r := range member.Roles {
		roleSet[r] = struct{}{}
	}
	for _, ar := range p.AllowedRoles {
		if _, ok := roleSet[ar.RoleID]; ok {
			return member, true
		}
	}

	return member, false
}

func (q *Bot) sendDM(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.EphemeralThink(s, i); err != nil {
		return nil
	}

	// Locate post by CustomID or fallback to message -> post map (legacy)
	var postKey string
	cid := i.MessageComponentData().CustomID
	if strings.HasPrefix(cid, sendDM+":") {
		postKey = strings.TrimPrefix(cid, sendDM+":")
	} else if strings.HasPrefix(cid, showImage+":") { // if coming from show button
		postKey = strings.TrimPrefix(cid, showImage+":")
	}
	if postKey == "" {
		q.mu.Lock()
		postKey = q.msgToPost[i.Message.ID]
		q.mu.Unlock()
	}
	if postKey == "" {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "I couldn't link this button to a post.")
	}
	p, err := q.db.ReadPostByExternalID(postKey)
	if err != nil || p == nil || len(p.Images) == 0 {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Couldn't load the image for this post.", err)
	}

	member, allowed := q.isAllowedToView(s, i, p)
	if !allowed {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "You don't have access to view this image.")
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

	ch, err := s.UserChannelCreate(userID)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction,
			"I couldn't open your DMs. Please allow DMs from this server and try again.", err)
	}

	if len(p.Images) == 0 {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "No image data available to DM.")
	}

	for idx, img := range p.Images {
		dmEmbed := &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeImage,
			Description: p.Description,
			Timestamp:   p.Timestamp.Format(time.RFC3339),
			Title:       p.Title,
		}
		if idx > 0 {
			dmEmbed.Title = fmt.Sprintf("%s (%d/%d)", p.Title, idx+1, len(p.Images))
			dmEmbed.Description = ""
		}
		if dmEmbed.Description == "" && idx == 0 {
			dmEmbed.Description = "New image posted!"
		}

		if p.Author != nil {
			if du := p.Author.ToDiscord(); du != nil {
				dmEmbed.Author = &discordgo.MessageEmbedAuthor{
					Name:    du.DisplayName(),
					IconURL: du.AvatarURL("64"),
					URL:     "https://discord.com/users/" + du.ID,
				}
			}
		}

		tooLarge := false
		for _, blob := range img.Blobs {
			if blob.Size > units.DiscordLimit || len(blob.Data) > units.DiscordLimit {
				tooLarge = true
				break
			}
		}

		if tooLarge {
			url, fbEmbed, ferr := q.fallbackToLink(i, p, idx)
			if ferr != nil {
				log.Error("Failed to upload image for fallback", "index", idx, "error", ferr)
				continue
			}

			content := fmt.Sprintf("📎 Image %d/%d was too large. Link: %s", idx+1, len(p.Images), url)
			if len(img.Blobs) > 0 && strings.Contains(utils.ContentType(img.Blobs[0].Data), "video/") {
				content = fmt.Sprintf("📎 Video %d/%d available. Link: %s", idx+1, len(p.Images), url)
			}

			_, sendErr := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
				Content: content,
				Embeds:  []*discordgo.MessageEmbed{fbEmbed},
				Components: []discordgo.MessageComponent{
					handlers.Components[handlers.DeleteButton],
				},
			})
			if sendErr != nil {
				log.Error("Failed to send fallback DM", "error", sendErr)
			}
			continue
		}

		memberExif := types.ToMemberExif(member)
		encoder := exif.NewEncoder(memberExif)
		var imageReaders []io.Reader

		for _, blob := range img.Blobs {
			imgBlob, err := q.db.GetImageBlob(blob.ID)
			if err != nil {
				log.Error("Failed to get blob data", "id", blob.ID, "error", err)
				continue
			}

			if int64(len(imgBlob.Data)) > units.DiscordLimit || imgBlob.Size > units.DiscordLimit {
				log.Warn("Skipping DM image due to size fetched from DB", "id", blob.ID)
				continue
			}

			if img.HasVideo() {
				imageReaders = append(imageReaders, bytes.NewReader(imgBlob.Data))
			} else {
				if imgBlob.ContentType == "image/png" {
					var buffer bytes.Buffer
					if err := encoder.EncodeReader(&buffer, bytes.NewReader(imgBlob.Data)); err != nil {
						log.Error("Failed to encode png", "error", err)
						imageReaders = append(imageReaders, bytes.NewReader(imgBlob.Data)) // fallback to original
					} else {
						imageReaders = append(imageReaders, &buffer)
					}
				} else {
					imageReaders = append(imageReaders, bytes.NewReader(imgBlob.Data))
				}
			}
		}

		var whEdit discordgo.WebhookEdit
		if err := handlers.EmbedImages(&whEdit, dmEmbed, imageReaders, nil, compositor.Compositor(memberExif)); err != nil {
			log.Error("Failed to prepare DM embed", "error", err)
			continue
		}

		var contentStr string
		if whEdit.Content != nil {
			contentStr = *whEdit.Content
		}

		_, err = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
			Content:    contentStr,
			Files:      whEdit.Files,
			Embeds:     *whEdit.Embeds,
			Components: []discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]},
		})
		if err != nil {
			log.Error("Failed to send DM", "error", err)
		}
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.Pointer("Sent! Check your DMs 📬"),
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{discordgo.Button{
					Label:    "Go to DM",
					Style:    discordgo.LinkButton,
					Disabled: false,
					Emoji:    &discordgo.ComponentEmoji{Name: "📩"},
					URL:      fmt.Sprintf("https://discord.com/channels/@me/%s", ch.ID),
					CustomID: "",
				}},
			}},
	})

	return nil
}

// fallbackToLink performs the upload of the image to the configured bucket and returns a public URL and a basic embed.
// Message sending/editing is intentionally left to the caller so this can be used by both showImage and sendDM paths.
func (q *Bot) fallbackToLink(i *discordgo.InteractionCreate, p *types.Post, imgIndex int) (url string, embed *discordgo.MessageEmbed, err error) {
	if p == nil || len(p.Images) == 0 {
		log.Error("fallbackToLink called with nil image")
		return "", nil, fmt.Errorf("no image found to upload")
	}

	if imgIndex < 0 || imgIndex >= len(p.Images) {
		imgIndex = 0
	}
	targetImg := p.Images[imgIndex]

	log.Debug("Falling back to link for post", "postKey", p.PostKey, "index", imgIndex)

	if q.bucket == nil {
		log.Error("fallbackToLink called but no bucket configured")
		return "", nil, fmt.Errorf("no bucket configured for fallback")
	}

	var data []byte
	if len(targetImg.Blobs) > 0 {
		if targetImg.Blobs[0].Data != nil {
			data = targetImg.Blobs[0].Data
		} else {
			if blob, err := q.db.GetImageBlob(targetImg.Blobs[0].ID); err == nil {
				data = blob.Data
			}
		}
	} else if len(targetImg.Thumbnail) > 0 {
		data = targetImg.Thumbnail
	}
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no image data available to upload")
	}

	contentType := utils.ContentType(data)
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		data, err = q.encodeExif(data, i.Member)
		if err != nil {
			log.Error("error encoding exif for fallback upload", "error", err)
			return "", nil, fmt.Errorf("failed to encode image for upload: %w", err)
		}
	case "video/mp4", "video/x-m4v", "video/x-msvideo", "video/x-flv":
	default:
		contentType = "image/png"
	}

	ts := time.Now().UTC().Format("20060102-150405")
	ext := map[string]string{
		"image/jpeg":      "jpg",
		"image/png":       "png",
		"image/webp":      "webp",
		"image/gif":       "gif",
		"video/mp4":       "mp4",
		"video/x-m4v":     "m4v",
		"video/x-msvideo": "avi",
		"video/x-flv":     "flv",
	}[contentType]
	if ext == "" {
		ext = "png"
	}
	key := fmt.Sprintf("%s/%s.%s", p.PostKey, ts, ext)

	url, err = q.bucket.Upload(q.context, key, data, contentType)
	if err != nil {
		return "", nil, fmt.Errorf("failed to upload image for fallback: %w", err)
	}

	embed = &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeImage,
		Description: p.Description,
		Timestamp:   p.Timestamp.Format(time.RFC3339),
		Title:       p.Title,
	}
	if embed.Description == "" {
		embed.Description = "New image posted!"
	}
	if p.Author != nil {
		if du := p.Author.ToDiscord(); du != nil {
			embed.Author = &discordgo.MessageEmbedAuthor{
				Name:    du.DisplayName(),
				IconURL: du.AvatarURL("64"),
				URL:     "https://discord.com/users/" + du.ID,
			}
		}
	}

	return url, embed, nil
}

func (q *Bot) encodeExif(data []byte, member *discordgo.Member) ([]byte, error) {
	memberExif := types.ToMemberExif(member)
	if memberExif == nil {
		log.Warn("no member info available for exif")
		return nil, errors.New("no member info available for exif")
	}
	encoder := exif.NewEncoder(memberExif)
	var buffer bytes.Buffer
	err := encoder.EncodeReader(&buffer, bytes.NewReader(data))
	if err != nil {
		log.Error("error encoding exif", "error", err)
		return nil, err
	}

	return buffer.Bytes(), nil
}
func (q *Bot) SendDirectMessage(userID, postKey string) error {
	s := q.botSession
	if s == nil {
		return fmt.Errorf("session not ready")
	}

	p, err := q.db.ReadPostByExternalID(postKey)
	if err != nil || p == nil || len(p.Images) == 0 {
		return fmt.Errorf("couldn't load post: %w", err)
	}

	ch, err := s.UserChannelCreate(userID)
	if err != nil {
		return fmt.Errorf("couldn't create DM channel: %w", err)
	}

	for idx, img := range p.Images {
		dmEmbed := &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeImage,
			Description: p.Description,
			Timestamp:   p.Timestamp.Format(time.RFC3339),
			Title:       p.Title,
		}
		if idx > 0 {
			dmEmbed.Title = fmt.Sprintf("%s (%d/%d)", p.Title, idx+1, len(p.Images))
			dmEmbed.Description = ""
		}
		if dmEmbed.Description == "" && idx == 0 {
			dmEmbed.Description = "New image posted!"
		}

		if p.Author != nil {
			if du := p.Author.ToDiscord(); du != nil {
				dmEmbed.Author = &discordgo.MessageEmbedAuthor{
					Name:    du.DisplayName(),
					IconURL: du.AvatarURL("64"),
					URL:     "https://discord.com/users/" + du.ID,
				}
			}
		}

		// Check size
		tooLarge := false
		for _, blob := range img.Blobs {
			if blob.Size > units.DiscordLimit || len(blob.Data) > units.DiscordLimit {
				tooLarge = true
				break
			}
		}

		if tooLarge {
			log.Warn("Skipping DM image due to size", "post", postKey, "index", idx)
			continue
		}

		var imageReaders []io.Reader
		for _, blob := range img.Blobs {
			imgBlob, err := q.db.GetImageBlob(blob.ID)
			if err != nil {
				continue
			}

			if int64(len(imgBlob.Data)) > units.DiscordLimit || imgBlob.Size > units.DiscordLimit {
				log.Warn("Skipping DM image due to size fetched from DB", "post", postKey, "index", idx)
				continue
			}
			imageReaders = append(imageReaders, bytes.NewReader(imgBlob.Data))
		}

		var whEdit discordgo.WebhookEdit
		if err := handlers.EmbedImages(&whEdit, dmEmbed, imageReaders, nil, compositor.Passthrough); err != nil {
			log.Error("Failed to prepare DM embed", "error", err)
			continue
		}

		var contentStr string
		if whEdit.Content != nil {
			contentStr = *whEdit.Content
		}

		_, err = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
			Content: contentStr,
			Files:   whEdit.Files,
			Embeds:  *whEdit.Embeds,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
