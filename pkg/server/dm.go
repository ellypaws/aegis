package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) handlePostDM(c echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "post id required"})
	}

	user := GetUserFromContext(c)
	if user == nil || user.UserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	post, err := s.db.ReadPostByExternalID(postID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}
	if !canAccessPost(user, post) {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Access denied"})
	}

	err = s.bot.SendDM(user.UserID, postID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "sent"})
}
