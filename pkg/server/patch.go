package server

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"

	"drigo/pkg/types"
)

func (s *Server) handlePatchPost(c echo.Context) error {
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
		log.Error("Failed to find post for patch", "id", idStr, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}

	// Parse multipart form (32MB max memory)
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to parse multipart form"})
	}

	// Read form values
	title := c.FormValue("title")
	description := c.FormValue("description")
	rolesStr := c.FormValue("roles")
	channelsStr := c.FormValue("channels")

	// Update scalar fields on the post
	post.Title = title
	post.Description = description
	post.ChannelID = channelsStr

	// Focus
	if v, err := strconv.ParseFloat(c.FormValue("focusX"), 64); err == nil {
		post.FocusX = &v
	}
	if v, err := strconv.ParseFloat(c.FormValue("focusY"), 64); err == nil {
		post.FocusY = &v
	}

	// Handle Roles
	var allowedRoles []types.Allowed
	for _, rid := range strings.Split(rolesStr, ",") {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		role, err := s.bot.Session().State.Role(s.config.GuildID, rid)
		if err == nil {
			allowedRoles = append(allowedRoles, types.Allowed{
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
			})
		} else {
			allowedRoles = append(allowedRoles, types.Allowed{RoleID: rid})
		}
	}
	post.AllowedRoles = allowedRoles

	form := c.Request().MultipartForm
	files := form.File["images"]
	if len(files) == 0 {
		files = form.File["image"]
	}

	// If we have new files, verify and append
	if len(files) > 0 {
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

			// Append new image
			post.Images = append(post.Images, types.Image{
				PostID: post.ID, // Link to this post
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
	} else {
		for i := range post.Images {
			post.Images[i].Blobs = nil
		}
	}

	// UpdatePost in sqlite/posts.go should be robust enough.
	if err := s.db.UpdatePost(post); err != nil {
		log.Error("Failed to update post", "id", post.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update post"})
	}
	s.getPostCache.Reset()

	// Read again to return fully hydrated post
	updated, err := s.db.ReadPostByExternalID(idStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error reading updated post"})
	}

	log.Info("Post updated", "id", idStr, "by", user.Username)
	return c.JSON(http.StatusOK, updated)
}
