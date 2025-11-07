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
		MinValues:   utils.Pointer(0),
		MaxValues:   25,
		Required:    utils.Pointer(false),
		Disabled:    false,
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
		MinValues:   utils.Pointer(0),
		MaxValues:   25,
		Required:    utils.Pointer(false),
		Disabled:    false,
	},
}

// buildShowActionsRow returns the two-button row (Show image + Send to DMs) with embedded postKey.
func buildShowActionsRow(postKey string) discordgo.ActionsRow {
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

type MemberExif struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	Nitro       bool   `json:"nitro,omitempty"`
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
	if err != nil || p == nil || p.Image == nil {
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
			url, fbEmbed, err := q.fallbackToLink(i, p)
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
	if p.Image.HasVideo() {
		if err := handlers.EmbedImages(webhookEdit, embed, p.Image.Readers(), nil, compositor.Passthrough); err != nil {
			return fmt.Errorf("error creating image embed: %w", err)
		}
		return nil
	}

	memberExif := q.memberExif(member)
	encoder := exif.NewEncoder(memberExif)
	var images []io.Reader
	for _, blob := range p.Image.Blobs {
		var buffer bytes.Buffer
		err := encoder.EncodeReader(&buffer, bytes.NewReader(blob.Data))
		if err != nil {
			return fmt.Errorf("error encoding: %w", err)
		}
		images = append(images, &buffer)
	}

	if err := handlers.EmbedImages(webhookEdit, embed, images, nil, compositor.Compositor(memberExif)); err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}
	return nil
}

func (q *Bot) memberExif(member *discordgo.Member) *MemberExif {
	var memberExif *MemberExif
	if member != nil && member.User != nil {
		memberExif = &MemberExif{
			ID:          member.User.ID,
			Username:    member.User.Username,
			DisplayName: member.DisplayName(),
			Nitro:       member.User.PremiumType != 0,
		}
	} else {
		log.Warn("no member info available for exif")
	}
	return memberExif
}

func (q *Bot) isAllowedToView(s *discordgo.Session, i *discordgo.InteractionCreate, p *types.Post) (*discordgo.Member, bool) {
	var member *discordgo.Member
	var allowed bool
	if len(p.AllowedRoles) > 0 {
		member = i.Member
		if member == nil {
			if i.GuildID == "" {
				log.Warn("no guild id found in interaction")
				return nil, false
			}
			user := utils.GetUser(i.Member, i.User)
			if user == nil {
				log.Warn("Member or User is nil on interaction")
				return nil, false
			}
			if m, err := s.State.Member(i.GuildID, user.ID); err == nil {
				member = m
			} else if m, err := s.GuildMember(i.GuildID, user.ID); err == nil {
				member = m
			}
		}
		if member != nil {
			roleSet := map[string]struct{}{}
			for _, r := range member.Roles {
				roleSet[r] = struct{}{}
			}
			for _, ar := range p.AllowedRoles {
				if _, ok := roleSet[ar.RoleID]; ok {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			return nil, false
		}
	} else {
		allowed = true
	}
	return member, allowed
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
	if err != nil || p == nil || p.Image == nil {
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

	if len(p.Image.Blobs) == 0 {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "No image data available to DM.")
	}

	dmEmbed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeImage,
		Description: p.Description,
		Timestamp:   p.Timestamp.Format(time.RFC3339),
		Title:       p.Title,
	}
	if dmEmbed.Description == "" {
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

	for _, blob := range p.Image.Blobs {
		if len(blob.Data) > units.DiscordLimit {
			url, fbEmbed, ferr := q.fallbackToLink(i, p)
			if ferr != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to upload image for fallback.", ferr)
			}

			content := fmt.Sprintf("📎 Your image was too large to send directly. Here's a link: %s", url)
			if strings.Contains(utils.ContentType(blob.Data), "video/") {
				content = fmt.Sprintf("📎 Your video is ready. Here's a link: %s", url)
			}
			msg, sendErr := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
				Content: content,
				Embeds:  []*discordgo.MessageEmbed{fbEmbed},
				Components: []discordgo.MessageComponent{
					handlers.Components[handlers.DeleteButton],
				},
			})
			if sendErr != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Uploaded, but failed to DM you the link.", sendErr)
			}

			if _, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.Pointer("Sent a link! Check your DMs 📬"),
				Components: &[]discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{discordgo.Button{
							Label:    "Go to DM",
							Style:    discordgo.LinkButton,
							Disabled: false,
							Emoji:    &discordgo.ComponentEmoji{Name: "📩"},
							URL:      fmt.Sprintf("https://discord.com/channels/@me/%s/%s", ch.ID, msg.ID),
							CustomID: "",
							SKUID:    "",
							ID:       0,
						}},
						ID: 0,
					}},
			}); editErr != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "DM sent, but I couldn't update the response.", editErr)
			}
			return nil
		}
	}

	var webhookEdit discordgo.WebhookEdit
	err = q.prepareEmbed(member, p, &webhookEdit, dmEmbed)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to prepare embed", err)
	}

	message, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    "📩 Here's your image — thanks for your support!",
		Files:      webhookEdit.Files,
		Embeds:     *webhookEdit.Embeds,
		Components: []discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]},
	})
	if err != nil {
		var restErr *discordgo.RESTError
		if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeRequestEntityTooLarge {
			url, fbEmbed, err := q.fallbackToLink(i, p)
			if err != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to upload image for fallback.", err)
			}

			msg, sendErr := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
				Content: fmt.Sprintf("📎 Your image was too large to send directly. Here's a link: %s", url),
				Embeds:  []*discordgo.MessageEmbed{fbEmbed},
				Components: []discordgo.MessageComponent{
					handlers.Components[handlers.DeleteButton],
				},
			})
			if sendErr != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Uploaded, but failed to DM you the link.", sendErr)
			}

			if _, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.Pointer("Sent a link! Check your DMs 📬"),
				Components: &[]discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{discordgo.Button{
							Label:    "Go to DM",
							Style:    discordgo.LinkButton,
							Disabled: false,
							Emoji:    &discordgo.ComponentEmoji{Name: "📩"},
							URL:      fmt.Sprintf("https://discord.com/channels/@me/%s/%s", ch.ID, msg.ID),
							CustomID: "",
							SKUID:    "",
							ID:       0,
						}},
						ID: 0,
					}},
			}); editErr != nil {
				return handlers.ErrorFollowupEphemeral(s, i.Interaction, "DM sent, but I couldn't update the response.", editErr)
			}
			return nil
		}
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send you a DM.", err)
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
					URL:      fmt.Sprintf("https://discord.com/channels/@me/%s/%s", ch.ID, message.ID),
					CustomID: "",
					SKUID:    "",
					ID:       0,
				}},
				ID: 0,
			}},
	})
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "DM sent, but I couldn't update the response.", err)
	}

	return nil
}

// fallbackToLink performs the upload of the image to the configured bucket and returns a public URL and a basic embed.
// Message sending/editing is intentionally left to the caller so this can be used by both showImage and sendDM paths.
func (q *Bot) fallbackToLink(i *discordgo.InteractionCreate, p *types.Post) (url string, embed *discordgo.MessageEmbed, err error) {
	if p == nil || p.Image == nil {
		log.Error("fallbackToLink called with nil image")
		return "", nil, fmt.Errorf("no image found to upload")
	}

	log.Debug("Falling back to link for post", "postKey", p.PostKey)

	if q.bucket == nil {
		log.Error("fallbackToLink called but no bucket configured")
		return "", nil, fmt.Errorf("no bucket configured for fallback")
	}

	var data []byte
	if len(p.Image.Blobs) > 0 && p.Image.Blobs[0].Data != nil {
		data = p.Image.Blobs[0].Data
	} else if len(p.Image.Thumbnail) > 0 {
		data = p.Image.Thumbnail
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
	memberExif := q.memberExif(member)
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
