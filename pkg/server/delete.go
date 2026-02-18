package server

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleDeletePost(c echo.Context) error {
	user := GetUserFromContext(c)
	if user == nil || !user.IsAdmin {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Double-check against DB
	userDB, err := s.db.UserByID(user.UserID)
	if err != nil || userDB == nil || !userDB.IsAdmin {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing id"})
	}

	post, err := s.db.ReadPostByExternalID(idStr)
	if err != nil {
		log.Error("Failed to find post for delete", "id", idStr, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}

	if err := s.db.DeletePost(post.ID); err != nil {
		log.Error("Failed to delete post", "id", post.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete post"})
	}

	s.getPostCache.Reset()

	log.Info("Post deleted", "id", idStr, "by", user.Username)
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
