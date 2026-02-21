package server

import (
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"

	"drigo/pkg/types"
)

func (s *Server) UpdateGuildCacheFromState(guildID string) {
	guild, err := s.bot.Session().State.Guild(guildID)
	if err != nil {
		// Try fetching directly if state misses
		guild, err = s.bot.Session().Guild(guildID)
		if err != nil {
			log.Warn("Failed to get guild for cache update", "error", err)
			return
		}
	}

	var cachedRoles []types.CachedRole
	for _, role := range guild.Roles {
		if role == nil || role.ID == "" {
			continue
		}
		cachedRoles = append(cachedRoles, types.CachedRole{
			ID:      role.ID,
			Name:    role.Name,
			Color:   role.Color,
			Managed: role.Managed,
		})
	}

	var cachedChannels []types.CachedChannel
	for _, ch := range guild.Channels {
		if ch == nil || ch.ID == "" || ch.Type == discordgo.ChannelTypeGuildVoice {
			continue
		}
		cachedChannels = append(cachedChannels, types.CachedChannel{
			ID:   ch.ID,
			Name: ch.Name,
			Type: int(ch.Type),
		})
	}

	// Update DB
	err1 := s.db.UpdateCachedRoles(cachedRoles)
	err2 := s.db.UpdateCachedChannels(cachedChannels)
	if err1 != nil || err2 != nil {
		log.Warn("Failed to update db cache", "errRoles", err1, "errChannels", err2)
	}

	// Update Flight Cache
	s.guildCache.Set(struct{}{}, &GuildData{
		Roles:    cachedRoles,
		Channels: cachedChannels,
		Name:     guild.Name, // Using this in transit, even if not persisted to settings table right now
	})
}
