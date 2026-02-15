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

	// Read form values — this handler accepts multipart/form-data
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

	// Parse postDate if provided
	// (We don't update timestamp on edit by default, but if the form sends it we can)

	// Handle Roles — rebuild AllowedRoles from the comma-separated role IDs
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

	// Clear the preloaded Image so UpdatePost won't try to re-save old blobs.
	// If the user uploads a new image we set a fresh Image below.
	// If not, setting nil tells UpdatePost to leave the existing image untouched.
	// But we do NOT want UpdatePost's nil-Image branch to DELETE the old image,
	// so we save the existing imageID and restore it only when no new image is provided.
	existingImage := post.Image
	post.Image = nil

	// Optional new image — if a new file is provided we replace it
	fileHeader, fileErr := c.FormFile("image")
	if fileErr == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open image"})
		}
		defer file.Close()

		imgBytes, err := io.ReadAll(file)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read image"})
		}

		// Fresh Image with ID=0 so UpdatePost creates a new row
		// (UpdatePost's nil-Image branch already removes old images for this post)
		post.Image = &types.Image{
			Blobs: []types.ImageBlob{
				{Index: 0, Data: imgBytes, ContentType: fileHeader.Header.Get("Content-Type"), Filename: fileHeader.Filename},
			},
		}
	} else {
		// No new image — restore the existing image pointer so UpdatePost
		// keeps it (the Image has a valid ID, no Blobs mutation needed).
		// We still nil out Blobs to prevent the re-create problem.
		if existingImage != nil {
			existingImage.Blobs = nil
			post.Image = existingImage
		}
	}

	// Save via UpdatePost which handles associations
	if err := s.db.UpdatePost(post); err != nil {
		log.Error("Failed to update post", "id", post.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update post"})
	}

	// Read again to return fully hydrated post
	updated, err := s.db.ReadPostByExternalID(idStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error reading updated post"})
	}

	log.Info("Post updated", "id", idStr, "by", user.Username)
	return c.JSON(http.StatusOK, updated)
}
