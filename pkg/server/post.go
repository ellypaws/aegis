package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/segmentio/ksuid"
	"golang.org/x/oauth2"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg/drigo"
	"drigo/pkg/types"
)

func getHost(c echo.Context) string {
	if c.Request().TLS != nil {
		return "https://" + c.Request().Host
	}

	return "http://" + c.Request().Host
}

func (s *Server) handleLogin(c echo.Context) error {
	redirectURI := getHost(c) + "/auth/callback"

	redirectURL := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify%%20guilds",
		s.bot.Session().State.User.ID, redirectURI)

	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (s *Server) handleCallback(c echo.Context) error {
	<-s.bot.Ready()
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing code"})
	}

	clientSecret := os.Getenv("CLIENT_SECRET")
	clientID := os.Getenv("CLIENT_ID")
	redirectURL := getHost(c) + "/auth/callback"

	endpoint := oauth2.Endpoint{
		AuthURL:  "https://discord.com/api/oauth2/authorize",
		TokenURL: "https://discord.com/api/oauth2/token",
	}

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"identify", "guilds"},
		Endpoint:     endpoint,
	}

	token, err := conf.Exchange(c.Request().Context(), code)
	if err != nil {
		log.Error("Failed to exchange token", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to exchange token"})
	}

	// Get User
	oauthClient := conf.Client(c.Request().Context(), token)
	resp, err := oauthClient.Get("https://discord.com/api/users/@me")
	if err != nil {
		log.Error("Failed to get user", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user"})
	}
	defer resp.Body.Close()

	var discordUser discordgo.User
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		log.Error("Failed to decode user", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode user"})
	}

	// Check if user is in the guild and has roles
	member, err := s.bot.Session().State.Member(s.config.GuildID, discordUser.ID)
	if err != nil {
		log.Error("Failed to get guild member", "user", discordUser.ID, "guild", s.config.GuildID, "error", err)
	}

	// Map roles
	var userRoles []types.Allowed
	var avatarURL string
	if member != nil {
		for _, roleID := range member.Roles {
			role, err := s.bot.Session().State.Role(s.config.GuildID, roleID)
			if err == nil {
				userRoles = append(userRoles, types.Allowed{
					RoleID: role.ID,
					Name:   role.Name,
					Color:  role.Color,
				})
			}
		}
		avatarURL = member.AvatarURL("")
	}

	var isAdmin bool
	if u, err := s.db.UserByID(discordUser.ID); err == nil && u != nil {
		isAdmin = u.IsAdmin
	} else {
		log.Info("User logging in not found in DB or not admin", "user", discordUser.Username, "id", discordUser.ID)
	}

	if avatarURL == "" {
		avatarURL = discordUser.AvatarURL("")
	}

	user := types.User{
		UserID:   discordUser.ID,
		Username: discordUser.Username,
		Avatar:   avatarURL,
		IsAdmin:  isAdmin,
	}

	// Generate JWT
	roleIDs := make([]string, len(userRoles))
	for i, r := range userRoles {
		roleIDs[i] = r.RoleID
	}

	if err := s.db.UpsertUser(&user); err != nil {
		log.Error("Failed to upsert user", "error", err)
	}

	if !user.IsAdmin {
		count, err := s.db.CountUsers()
		if err == nil && count == 1 {
			log.Info("Promoting first user to admin", "user", user.Username)
			if err := s.db.SetAdmin(user.UserID, true); err == nil {
				user.IsAdmin = true
			} else {
				log.Error("Failed to set admin", "error", err)
			}
		} else if !user.IsAdmin {
			log.Warn("User is not admin", "user", user.Username, "id", user.UserID)
		}
	}

	tokenString, err := GenerateToken(&user, roleIDs)
	if err != nil {
		log.Error("Failed to generate token", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	redirectTarget := fmt.Sprintf("http://localhost:5173/?token=%s", tokenString)
	return c.Redirect(http.StatusTemporaryRedirect, redirectTarget)
}

func (s *Server) handleCreatePost(c echo.Context) error {
	// Authorization check
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Parse multipart form
	// Expect fields: title, description, roles (comma separated ids), channels (comma separated ids)
	// Expect file: image
	title := c.FormValue("title")
	description := c.FormValue("description")
	rolesStr := c.FormValue("roles")
	channelsStr := c.FormValue("channels")

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing image"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open image"})
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read image"})
	}

	// Create Post
	postKey := ksuid.New().String()
	now := time.Now().UTC()

	// Handle Roles
	var allowedRoles []types.Allowed
	for _, rid := range strings.Split(rolesStr, ",") {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		// Verify role exists in guild using session
		role, err := s.bot.Session().State.Role(s.config.GuildID, rid)
		if err == nil {
			allowedRoles = append(allowedRoles, types.Allowed{
				RoleID:       role.ID,
				Name:         role.Name,
				Color:        role.Color,
				Permissions:  role.Permissions,
				Managed:      role.Managed,
				Mentionable:  role.Mentionable,
				Hoist:        role.Hoist,
				Position:     role.Position,
				Icon:         role.Icon,
				UnicodeEmoji: role.UnicodeEmoji,
				Flags:        int(role.Flags),
			})
		} else {
			// Fallback if not found in state
			allowedRoles = append(allowedRoles, types.Allowed{RoleID: rid})
		}
	}

	post := &types.Post{
		PostKey:     postKey,
		ChannelID:   channelsStr, // Storing multiple channel IDs? DB schema expects singular probably, or we just use first/origin. Let's assume origin for now is first selected.
		GuildID:     s.config.GuildID,
		Title:       title,
		Description: description,
		Timestamp:   now,
		IsPremium:   true,
		Image: &types.Image{
			Thumbnail: imgBytes, // Using full image as thumbnail for simplicity as we don't have resizer here yet
			Blobs: []types.ImageBlob{
				{Index: 0, Data: imgBytes, ContentType: fileHeader.Header.Get("Content-Type")},
			},
		},
		AllowedRoles: allowedRoles,
	}

	// Resolve Author from DB or Context
	if u, err := s.db.UserByID(user.UserID); err == nil && u != nil {
		post.AuthorID = u.ID
		post.Author = u
	} else {
		// Create incomplete user struct if not exists?
		post.Author = &types.User{
			UserID:   user.UserID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
		}
	}

	// Save to DB
	if err := s.db.CreatePost(post); err != nil {
		log.Error("Failed to create post in db", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create post"})
	}

	// Post to Discord Channels
	channelIDs := strings.Split(channelsStr, ",")

	// Prepare Embed
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Type:        discordgo.EmbedTypeImage,
		Timestamp:   now.Format(time.RFC3339),
		Description: description,
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://" + fileHeader.Filename,
		},
	}
	if embed.Description == "" {
		embed.Description = "New image posted!"
	}
	if post.Author != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name: post.Author.Username,
			// IconURL: ... we'd need avatar URL
		}
	}

	// Allowed roles text
	var sb strings.Builder
	sb.WriteString("> Allowed roles:\n")
	for _, r := range post.AllowedRoles {
		sb.WriteString("> <@&")
		sb.WriteString(r.RoleID)
		sb.WriteString(">\n")
	}

	// Components: Action Row with Link Button
	// TODO: Replace localhost with configured public URL
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:5173"
	}

	messageComponents := append([]discordgo.MessageComponent{
		discordgo.Button{
			Label: "View in Gallery",
			Style: discordgo.LinkButton,
			URL:   fmt.Sprintf("%s?post=%s", publicURL, postKey),
		},
	}, drigo.BuildShowActionsRow(postKey).Components...)

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: messageComponents,
		},
	}

	for _, chID := range channelIDs {
		chID = strings.TrimSpace(chID)
		if chID == "" {
			continue
		}

		// Send message with file
		files := []*discordgo.File{
			{
				Name:        fileHeader.Filename,
				ContentType: fileHeader.Header.Get("Content-Type"),
				Reader:      bytes.NewReader(imgBytes),
			},
		}

		_, err := s.bot.Session().ChannelMessageSendComplex(chID, &discordgo.MessageSend{
			Content:    sb.String(),
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Files:      files,
		})
		if err != nil {
			log.Error("Failed to send to channel", "channel", chID, "error", err)
		} else {
			log.Info("Posted to Discord channel", "channel", chID)
		}
	}

	return c.JSON(http.StatusOK, post)
}
