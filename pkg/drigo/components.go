package drigo

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

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

	if len(p.AllowedRoles) > 0 {
		allowed := false
		var member *discordgo.Member
		if i.Member != nil {
			member = i.Member
		} else if i.User != nil && i.GuildID != "" {
			// Try to fetch member from state or API
			if m, err := s.State.Member(i.GuildID, i.User.ID); err == nil {
				member = m
			} else if m, err := s.GuildMember(i.GuildID, i.User.ID); err == nil {
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
			return handlers.ErrorFollowupEphemeral(s, i.Interaction, "You don't have access to view this image.")
		}
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

	var images []io.Reader
	if len(p.Image.Blobs) > 0 {
		images = append(images, bytes.NewReader(p.Image.Blobs[0].Data))
	}
	if err := utils.EmbedImages(webhookEdit, embed, images, nil, compositor.Compositor()); err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, webhookEdit); err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to send response", err)
	}

	return nil
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

	if len(p.AllowedRoles) > 0 {
		allowed := false
		var member *discordgo.Member
		if i.Member != nil {
			member = i.Member
		} else if i.User != nil && i.GuildID != "" {
			if m, err := s.State.Member(i.GuildID, i.User.ID); err == nil {
				member = m
			} else if m, err := s.GuildMember(i.GuildID, i.User.ID); err == nil {
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
			return handlers.ErrorFollowupEphemeral(s, i.Interaction, "You don't have access to view this image.")
		}
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

	var imgReader io.Reader
	if len(p.Image.Blobs) > 0 {
		imgReader = bytes.NewReader(p.Image.Blobs[0].Data)
	}
	if imgReader == nil {
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

	var webhookEdit discordgo.WebhookEdit
	err = utils.EmbedImages(&webhookEdit, dmEmbed, []io.Reader{imgReader}, nil, compositor.Compositor())
	if err != nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Failed to prepare embed.", err)
	}

	message, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    "📩 Here's your image — thanks for your support!",
		Files:      webhookEdit.Files,
		Embeds:     *webhookEdit.Embeds,
		Components: []discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]},
	})
	if err != nil {
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
