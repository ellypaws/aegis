package server

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"

	"drigo/pkg/sqlite"
)

func (s *Server) handlePatchPost(c echo.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing id"})
	}

	// We need internal ID (uint) for PatchPost, but we have external ID (ksuid)
	// So first we read the post to get the internal ID
	post, err := s.db.ReadPostByExternalID(idStr)
	if err != nil {
		log.Error("Failed to find post for patch", "id", idStr, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}

	var body struct {
		Content   *string `json:"content"`
		ChannelID *string `json:"channelId"`
		// Add other fields as needed
	}

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid body"})
	}

	patch := sqlite.PostPatch{
		Content:   body.Content,
		ChannelID: body.ChannelID,
	}

	if err := s.db.PatchPost(post.ID, patch); err != nil {
		log.Error("Failed to patch post", "id", post.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to patch post"})
	}

	// Read again to return updated
	updated, err := s.db.ReadPost(post.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error reading updated post"})
	}

	return c.JSON(http.StatusOK, updated)
}
