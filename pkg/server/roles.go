package server

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg/types"
)

func roleToAllowed(role *discordgo.Role) types.Allowed {
	return types.Allowed{
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
	}
}

func (s *Server) parseAllowedRoles(rolesStr string) []types.Allowed {
	if strings.TrimSpace(rolesStr) == "" {
		return nil
	}

	roleByID := map[string]*discordgo.Role{}
	if guildRoles, err := s.bot.Session().GuildRoles(s.config.GuildID); err == nil {
		for _, role := range guildRoles {
			if role == nil || role.ID == "" {
				continue
			}
			roleByID[role.ID] = role
		}
		// Also update the cache since we just hit the API
		go s.UpdateGuildCacheFromState(s.config.GuildID)
	}

	seen := map[string]struct{}{}
	allowedRoles := make([]types.Allowed, 0)
	for _, rid := range strings.Split(rolesStr, ",") {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		if _, ok := seen[rid]; ok {
			continue
		}
		seen[rid] = struct{}{}

		if role, err := s.bot.Session().State.Role(s.config.GuildID, rid); err == nil && role != nil {
			allowedRoles = append(allowedRoles, roleToAllowed(role))
			continue
		}

		if role, ok := roleByID[rid]; ok {
			allowedRoles = append(allowedRoles, roleToAllowed(role))
			continue
		}

		allowedRoles = append(allowedRoles, types.Allowed{RoleID: rid})
	}

	return allowedRoles
}
