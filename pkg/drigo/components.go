package drigo

import (
	"bytes"
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
	// A small row with only a DM button for ephemeral follow-ups
	sendDM: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Send to DMs",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				Emoji:    &discordgo.ComponentEmoji{Name: "📩"},
				CustomID: sendDM,
			},
		},
	},
}

func (q *Bot) showImage(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.EphemeralThink(s, i); err != nil {
		return nil
	}

	// Find post by message mapping
	q.mu.Lock()
	postKey := q.msgToPost[i.Message.ID]
	q.mu.Unlock()
	if postKey == "" {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "I couldn't link this button to a post.")
	}

	p, err := q.db.ReadPostByExternalID(postKey)
	if err != nil || p == nil || p.Image == nil {
		return handlers.ErrorFollowupEphemeral(s, i.Interaction, "Couldn't load the image for this post.", err)
	}

	// Authorization: user must have any of the AllowedRoles
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
			// Build a set of role IDs on the member
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
		Components: &[]discordgo.MessageComponent{components[sendDM]},
	}

	// Build image readers from DB bytes
	imgs := []io.Reader{}
	if len(p.Image.Blobs) > 0 {
		imgs = append(imgs, bytes.NewReader(p.Image.Blobs[0].Data))
	}
	if err := utils.EmbedImages(webhookEdit, nil, imgs, nil, compositor.Compositor()); err != nil {
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

	// Locate post by message
	q.mu.Lock()
	postKey := q.msgToPost[i.Message.ID]
	q.mu.Unlock()
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

	message, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: "📩 Here's your image — thanks for your support!",
		Files:   []*discordgo.File{{Name: "image.png", ContentType: "image/png", Reader: imgReader}},
		Components: []discordgo.MessageComponent{
			discordgo.MediaGallery{
				Items: []discordgo.MediaGalleryItem{{Media: discordgo.UnfurledMediaItem{URL: "attachment://image.png"}}},
			},
			handlers.Components[handlers.DeleteButton],
		},
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
					URL:      fmt.Sprintf("https://discord.com/channels/@me/%s/%s", s.State.User.ID, message.ID),
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
