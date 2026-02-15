package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"drigo/pkg/sqlite"
)

func (s *Server) handleGetSettings(c echo.Context) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch settings"})
	}
	return c.JSON(http.StatusOK, settings)
}

func (s *Server) handleUpdateSettings(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
	}

	var newSettings sqlite.Settings
	if err := c.Bind(&newSettings); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	updated, err := s.db.UpdateSettings(newSettings)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update settings"})
	}

	return c.JSON(http.StatusOK, updated)
}
