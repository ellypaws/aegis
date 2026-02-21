package server

import (
	"net/http"
	"strconv"

	"drigo/pkg/types"
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
)

type sortOption struct {
	limit  int
	offset int
	sort   string
}

func (s *Server) handleGetPosts(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit
	sort := c.QueryParam("sort")

	posts, err := s.getPostCache.Get(sortOption{limit, offset, sort})
	if err != nil {
		log.Error("Failed to list posts", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list posts"})
	}

	return c.JSON(http.StatusOK, posts)
}

func (s *Server) handleGetPost(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing id"})
	}

	// Assuming id is the external key (ksuid)
	post, err := s.db.ReadPostByExternalID(id)
	if err != nil {
		log.Error("Failed to read post", "id", id, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}

	return c.JSON(http.StatusOK, post)
}

func (s *Server) handleGetRoles(c echo.Context) error {
	roles, err := s.db.AllowedRoles()
	if err != nil {
		log.Error("Failed to get allowed roles", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get roles"})
	}

	return c.JSON(http.StatusOK, roles)
}

func (s *Server) handleGetUploadConfig(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	guildID := s.config.GuildID
	guild, err := s.bot.Session().Guild(guildID)
	if err != nil {
		log.Error("Failed to get guild", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get guild"})
	}

	roles := guild.Roles
	if len(roles) == 0 {
		log.Warn("No roles found in guild", "guildID", guildID)
	}

	channels, err := s.bot.Session().GuildChannels(guildID)
	if err != nil {
		log.Error("Failed to get channels", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get channels"})
	}
	if len(channels) == 0 {
		log.Warn("No channels found in guild", "guildID", guildID)
	}

	type ChannelResp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type int    `json:"type"`
	}

	var channelResps []ChannelResp
	var cachedChannels []types.CachedChannel
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildVoice {
			continue
		}
		channelResps = append(channelResps, ChannelResp{
			ID:   ch.ID,
			Name: ch.Name,
			Type: int(ch.Type),
		})
		cachedChannels = append(cachedChannels, types.CachedChannel{
			ID:   ch.ID,
			Name: ch.Name,
			Type: int(ch.Type),
		})
	}

	var cachedRoles []types.CachedRole
	for _, role := range roles {
		cachedRoles = append(cachedRoles, types.CachedRole{
			ID:      role.ID,
			Name:    role.Name,
			Color:   role.Color,
			Managed: role.Managed,
		})
	}

	// Update DB and Memory asynchronously since this is user-facing fast path
	go func() {
		_ = s.db.UpdateCachedRoles(cachedRoles)
		_ = s.db.UpdateCachedChannels(cachedChannels)
		s.guildCache.Set(struct{}{}, &GuildData{
			Roles:    cachedRoles,
			Channels: cachedChannels,
			Name:     guild.Name,
		})
	}()

	return c.JSON(http.StatusOK, map[string]any{
		"roles":      roles,
		"channels":   channelResps,
		"guild_name": guild.Name,
	})
}
