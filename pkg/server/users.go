package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) handleGetUsers(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
	}

	users, err := s.db.GetAllUsers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch users"})
	}
	return c.JSON(http.StatusOK, users)
}

func (s *Server) handleToggleAdmin(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized"})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing user ID"})
	}

	var req struct {
		IsAdmin bool `json:"isAdmin"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := s.db.SetAdmin(id, req.IsAdmin); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update admin status"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"success": true, "isAdmin": req.IsAdmin})
}
