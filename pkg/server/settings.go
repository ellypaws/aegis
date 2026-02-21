package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"drigo/pkg/types"
)

func (s *Server) handleGetSettings(c echo.Context) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch settings"})
	}

	guildData, _ := s.guildCache.Get(struct{}{})

	return c.JSON(http.StatusOK, map[string]any{
		"ID":               settings.ID,
		"CreatedAt":        settings.CreatedAt,
		"UpdatedAt":        settings.UpdatedAt,
		"DeletedAt":        settings.DeletedAt,
		"hero_title":       settings.HeroTitle,
		"hero_subtitle":    settings.HeroSubtitle,
		"hero_description": settings.HeroDescription,
		"public_access":    settings.PublicAccess,
		"theme":            settings.Theme,
		"roles":            guildData.Roles,
		"channels":         guildData.Channels,
		"guild_name":       guildData.Name,
	})
}

func (s *Server) handleUpdateSettings(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
	}

	var newSettings types.Settings
	if err := c.Bind(&newSettings); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	updated, err := s.db.UpdateSettings(newSettings)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update settings"})
	}

	return c.JSON(http.StatusOK, updated)
}
