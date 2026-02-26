package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	"github.com/labstack/echo/v4"
	"github.com/segmentio/ksuid"
	"golang.org/x/image/draw"

	"github.com/charmbracelet/log"
	"golang.org/x/oauth2"

	"drigo/pkg/drigo"
	"drigo/pkg/types"
	"drigo/pkg/units"
)

func getHost(c echo.Context) string {
	return "https://" + c.Request().Host
}

func (s *Server) processThumbnail(ctx context.Context, data []byte, blur bool, postKey string) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode: %w", err)
	}

	if blur {
		img = imaging.Blur(img, 32.0) // Apply Gaussian blur
	}

	quality := 100
	var outBytes []byte

	// iteratively compress quality
	for quality >= 80 {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, webp.Options{Quality: quality}); err != nil {
			return nil, "", fmt.Errorf("encode webp: %w", err)
		}
		if buf.Len() <= units.DiscordLimit {
			outBytes = buf.Bytes()
			break
		}
		quality -= 5
	}

	// reduce resolution if still too large at 80 quality
	if len(outBytes) == 0 {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, webp.Options{Quality: 80}); err != nil {
			return nil, "", fmt.Errorf("encode webp: %w", err)
		}
		outBytes = buf.Bytes()

		for len(outBytes) > units.DiscordLimit {
			bounds := img.Bounds()
			newWidth := bounds.Dx() * 9 / 10 // 90% resolution reduction
			if newWidth < 10 {
				break // give up if too small
			}
			ratio := float64(bounds.Dx()) / float64(bounds.Dy())
			newHeight := int(float64(newWidth) / ratio)
			if newHeight < 1 {
				newHeight = 1
			}

			dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
			draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
			img = dst

			var resizeBuf bytes.Buffer
			if err := webp.Encode(&resizeBuf, img, webp.Options{Quality: 80}); err != nil {
				return nil, "", fmt.Errorf("encode webp: %w", err)
			}
			outBytes = resizeBuf.Bytes()
		}
	}

	// S3 upload if still > 10MB or we want to offload (we check for DiscordLimit)
	if len(outBytes) > units.DiscordLimit && s.bucket != nil {
		ts := time.Now().UTC().Format("20060102-150405")
		key := fmt.Sprintf("%s/%s_thumb.webp", postKey, ts)
		url, err := s.bucket.Upload(ctx, key, outBytes, "image/webp")
		if err != nil {
			log.Error("Failed to upload thumbnail to S3", "error", err)
			return outBytes, "", nil // fallback to local bytes
		}
		return outBytes, url, nil
	}

	return outBytes, "", nil
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

	member, err := s.bot.Session().State.Member(s.config.GuildID, discordUser.ID)
	if err != nil {
		member, err = s.bot.Session().GuildMember(s.config.GuildID, discordUser.ID)
		if err != nil {
			log.Error("Failed to get guild member", "user", discordUser.ID, "guild", s.config.GuildID, "error", err)
		}
	}

	var roles []*discordgo.Role
	var avatarHash string
	if member != nil {
		for _, rid := range member.Roles {
			if r, err := s.bot.Session().State.Role(s.config.GuildID, rid); err == nil {
				roles = append(roles, r)
			}
		}
		// Use member avatar if available, otherwise fall back to user avatar
		if member.Avatar != "" {
			avatarHash = member.Avatar
		}
	}

	var isAdmin bool
	if u, err := s.db.UserByID(discordUser.ID); err == nil && u != nil {
		isAdmin = u.IsAdmin
	} else {
		log.Info("User logging in not found in DB or not admin", "user", discordUser.Username, "id", discordUser.ID)
	}

	// Fall back to user avatar hash if no member avatar
	if avatarHash == "" {
		avatarHash = discordUser.Avatar
	}

	user := types.User{
		UserID:   discordUser.ID,
		Username: discordUser.Username,
		Avatar:   avatarHash,
		Banner:   discordUser.Banner,
		IsAdmin:  isAdmin,
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
		}
	}

	tokenString, err := GenerateToken(&user, roles)
	if err != nil {
		log.Error("Failed to generate token", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	redirectTarget := fmt.Sprintf("/?token=%s", tokenString)
	return c.Redirect(http.StatusTemporaryRedirect, redirectTarget)
}

func (s *Server) handleCreatePost(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	userDB, err := s.db.UserByID(user.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user"})
	}
	if !userDB.IsAdmin {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Parse multipart form (32MB max memory)
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to parse multipart form"})
	}

	title := c.FormValue("title")
	description := c.FormValue("description")
	rolesStr := c.FormValue("roles")
	channelsStr := c.FormValue("channels")

	form := c.Request().MultipartForm
	files := form.File["images"] // Expecting "images" key for multiple files
	thumbFiles := form.File["thumbnail"]

	// Fallback to "image" if "images" is empty (backward compatibility)
	if len(files) == 0 {
		files = form.File["image"]
	}

	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing images"})
	}

	postKey := ksuid.New().String()
	now := time.Now().UTC()

	// Use author-supplied date if provided
	if dateStr := c.FormValue("postDate"); dateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
			now = parsed.UTC()
		} else if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			now = parsed.UTC()
		}
	}

	// Handle Roles
	allowedRoles := s.parseAllowedRoles(rolesStr)

	// Parse focus position (percentage 0-100, default 50 = center)
	focusX := 50.0
	if v, err := strconv.ParseFloat(c.FormValue("focusX"), 64); err == nil {
		focusX = v
	}
	focusY := 50.0
	if v, err := strconv.ParseFloat(c.FormValue("focusY"), 64); err == nil {
		focusY = v
	}

	// Process Images
	var postImages []types.Image

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			log.Error("Failed to open uploaded file", "filename", fileHeader.Filename, "error", err)
			continue
		}

		imgBytes, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			log.Error("Failed to read uploaded file", "filename", fileHeader.Filename, "error", err)
			continue
		}

		postImages = append(postImages, types.Image{
			Blobs: []types.ImageBlob{
				{
					Index:       0,
					Data:        imgBytes,
					ContentType: fileHeader.Header.Get("Content-Type"),
					Filename:    fileHeader.Filename,
				},
			},
		})
	}

	var finalS3ThumbUrl string

	if len(thumbFiles) > 0 && len(postImages) > 0 {
		thumbFile, err := thumbFiles[0].Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to open thumbnail"})
		}
		thumbBytes, err := io.ReadAll(thumbFile)
		thumbFile.Close()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read thumbnail"})
		}

		thumbBytes, s3ThumbUrl, err := s.processThumbnail(c.Request().Context(), thumbBytes, false, postKey)
		if err != nil {
			log.Error("Failed to process uploaded thumbnail", "error", err)
		} else {
			postImages[0].Thumbnail = thumbBytes
			finalS3ThumbUrl = s3ThumbUrl
		}
	} else if len(postImages) > 0 {
		needsBlur := len(allowedRoles) > 0
		thumbBytes, s3ThumbUrl, err := s.processThumbnail(c.Request().Context(), postImages[0].Blobs[0].Data, needsBlur, postKey)
		if err != nil {
			log.Error("Failed to generate missing thumbnail", "error", err)
		} else {
			postImages[0].Thumbnail = thumbBytes
			finalS3ThumbUrl = s3ThumbUrl
		}
	}

	if len(postImages) == 0 {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process any images"})
	}

	post := &types.Post{
		PostKey:      postKey,
		ChannelID:    channelsStr,
		GuildID:      s.config.GuildID,
		Title:        title,
		Description:  description,
		Timestamp:    now,
		IsPremium:    true,
		FocusX:       &focusX,
		FocusY:       &focusY,
		Images:       postImages,
		AllowedRoles: allowedRoles,
	}

	// Resolve Author from DB or Context
	if u, err := s.db.UserByID(user.UserID); err == nil && u != nil {
		post.AuthorID = u.ID
		post.Author = u
	} else {
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
	s.getPostCache.Reset()

	// Post to Discord Channels
	channelIDs := strings.Split(channelsStr, ",")

	// Embed for thumbnail
	firstFile := files[0]
	thumbFilename := "thumb_" + firstFile.Filename + ".webp"

	thumbURL := "attachment://" + thumbFilename
	if finalS3ThumbUrl != "" {
		thumbURL = finalS3ThumbUrl
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Type:        discordgo.EmbedTypeImage,
		Timestamp:   now.Format(time.RFC3339),
		Description: description,
		Image: &discordgo.MessageEmbedImage{
			URL: thumbURL,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf(" +%d more", len(postImages)-1),
		},
	}
	if len(postImages) <= 1 {
		embed.Footer = nil
	}

	if embed.Description == "" {
		embed.Description = "New post!"
	}
	if post.Author != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name: post.Author.Username,
		}
	}

	var sb strings.Builder
	sb.WriteString("> Allowed roles:\n")
	for _, r := range post.AllowedRoles {
		sb.WriteString("> <@&")
		sb.WriteString(r.RoleID)
		sb.WriteString(">\n")
	}

	messageComponents := append([]discordgo.MessageComponent{
		discordgo.Button{
			Label: "View in Gallery",
			Style: discordgo.LinkButton,
			URL:   fmt.Sprintf("%s/post/%s", getHost(c), postKey),
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

		var discordFiles []*discordgo.File

		for _, img := range postImages {
			if len(img.Thumbnail) > 0 && finalS3ThumbUrl == "" {
				discordFiles = append(discordFiles, &discordgo.File{
					Name:        thumbFilename,
					ContentType: "image/webp",
					Reader:      bytes.NewReader(img.Thumbnail),
				})
				break // only attach one thumbnail
			}
		}

		_, err := s.bot.Session().ChannelMessageSendComplex(chID, &discordgo.MessageSend{
			Content:    sb.String(),
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Files:      discordFiles,
		})
		if err != nil {
			log.Error("Failed to send to channel", "channel", chID, "error", err)
		} else {
			log.Info("Posted to Discord channel", "channel", chID)
		}
	}

	return c.JSON(http.StatusOK, post)
}
