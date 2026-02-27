package server

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	var post *types.Post
	if numericID, parseErr := strconv.ParseUint(idStr, 10, 32); parseErr == nil {
		post, err = s.db.ReadPostWithBlobData(uint(numericID))
		if err != nil {
			post, err = s.db.ReadPostByExternalIDWithBlobData(idStr)
		}
	} else {
		post, err = s.db.ReadPostByExternalIDWithBlobData(idStr)
	}

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
	removeImageIDsStr := c.FormValue("removeImageIds")
	mediaOrderStr := c.FormValue("mediaOrder")
	clearThumbnail := c.FormValue("clearThumbnail") == "1" || strings.EqualFold(c.FormValue("clearThumbnail"), "true")

	// Update scalar fields on the post
	post.Title = title
	post.Description = description
	post.ChannelID = channelsStr

	// Use author-supplied date if provided
	if dateStr := c.FormValue("postDate"); dateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
			post.Timestamp = parsed.UTC()
		} else if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			post.Timestamp = parsed.UTC()
		}
	}

	// Focus
	if v, err := strconv.ParseFloat(c.FormValue("focusX"), 64); err == nil {
		post.FocusX = &v
	}
	if v, err := strconv.ParseFloat(c.FormValue("focusY"), 64); err == nil {
		post.FocusY = &v
	}

	// Handle Roles
	post.AllowedRoles = s.parseAllowedRoles(rolesStr)

	form := c.Request().MultipartForm
	files := form.File["images"]
	if len(files) == 0 {
		files = form.File["image"]
	}
	thumbFiles := form.File["thumbnail"]

	removedImageIDs := make(map[uint]struct{})
	for _, idToken := range strings.Split(removeImageIDsStr, ",") {
		idToken = strings.TrimSpace(idToken)
		if idToken == "" {
			continue
		}
		parsed, parseErr := strconv.ParseUint(idToken, 10, 32)
		if parseErr != nil {
			continue
		}
		removedImageIDs[uint(parsed)] = struct{}{}
	}

	newImages := make([]types.Image, 0, len(files))
	for _, fileHeader := range files {
		file, openErr := fileHeader.Open()
		if openErr != nil {
			log.Error("Failed to open uploaded file", "filename", fileHeader.Filename, "error", openErr)
			continue
		}

		imgBytes, readErr := io.ReadAll(file)
		file.Close()
		if readErr != nil {
			log.Error("Failed to read uploaded file", "filename", fileHeader.Filename, "error", readErr)
			continue
		}

		newImages = append(newImages, types.Image{
			PostID: post.ID,
			Blobs: []types.ImageBlob{
				{
					Index:       0,
					Data:        imgBytes,
					ContentType: fileHeader.Header.Get("Content-Type"),
					Filename:    fileHeader.Filename,
					Size:        int64(len(imgBytes)),
				},
			},
		})
	}

	copyImage := func(src types.Image) types.Image {
		copied := types.Image{
			Thumbnail: append([]byte(nil), src.Thumbnail...),
		}
		copied.Blobs = make([]types.ImageBlob, 0, len(src.Blobs))
		for i := range src.Blobs {
			blob := src.Blobs[i]
			copied.Blobs = append(copied.Blobs, types.ImageBlob{
				Index:       blob.Index,
				Data:        append([]byte(nil), blob.Data...),
				ContentType: blob.ContentType,
				Filename:    blob.Filename,
				Size:        blob.Size,
			})
		}
		return copied
	}

	existingByID := make(map[uint]types.Image, len(post.Images))
	for i := range post.Images {
		existingByID[post.Images[i].ID] = post.Images[i]
	}

	finalImages := make([]types.Image, 0, len(post.Images)+len(newImages))
	usedExisting := make(map[uint]struct{})
	usedNew := make(map[int]struct{})

	mediaOrderTokens := strings.Split(mediaOrderStr, ",")
	if strings.TrimSpace(mediaOrderStr) != "" {
		for _, token := range mediaOrderTokens {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}

			if strings.HasPrefix(token, "e:") {
				raw := strings.TrimSpace(strings.TrimPrefix(token, "e:"))
				parsed, parseErr := strconv.ParseUint(raw, 10, 32)
				if parseErr != nil {
					continue
				}
				id := uint(parsed)
				if _, removed := removedImageIDs[id]; removed {
					continue
				}
				img, ok := existingByID[id]
				if !ok {
					continue
				}
				if _, alreadyUsed := usedExisting[id]; alreadyUsed {
					continue
				}
				finalImages = append(finalImages, copyImage(img))
				usedExisting[id] = struct{}{}
				continue
			}

			if strings.HasPrefix(token, "n:") {
				raw := strings.TrimSpace(strings.TrimPrefix(token, "n:"))
				idx, parseErr := strconv.Atoi(raw)
				if parseErr != nil || idx < 0 || idx >= len(newImages) {
					continue
				}
				if _, alreadyUsed := usedNew[idx]; alreadyUsed {
					continue
				}
				finalImages = append(finalImages, newImages[idx])
				usedNew[idx] = struct{}{}
			}
		}

		for i := range post.Images {
			img := post.Images[i]
			if _, removed := removedImageIDs[img.ID]; removed {
				continue
			}
			if _, alreadyUsed := usedExisting[img.ID]; alreadyUsed {
				continue
			}
			finalImages = append(finalImages, copyImage(img))
		}
		for i := range newImages {
			if _, alreadyUsed := usedNew[i]; alreadyUsed {
				continue
			}
			finalImages = append(finalImages, newImages[i])
		}
	} else {
		for i := range post.Images {
			img := post.Images[i]
			if _, removed := removedImageIDs[img.ID]; removed {
				continue
			}
			finalImages = append(finalImages, copyImage(img))
		}
		finalImages = append(finalImages, newImages...)
	}

	if len(finalImages) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Post must contain at least one media item"})
	}

	if len(thumbFiles) > 0 {
		thumbFile, openErr := thumbFiles[0].Open()
		if openErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to open thumbnail"})
		}
		thumbBytes, readErr := io.ReadAll(thumbFile)
		thumbFile.Close()
		if readErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read thumbnail"})
		}
		finalImages[0].Thumbnail = thumbBytes
	} else if clearThumbnail {
		finalImages[0].Thumbnail = nil
	}

	post.Images = finalImages

	// UpdatePost in sqlite/posts.go should be robust enough.
	if err := s.db.UpdatePost(post); err != nil {
		log.Error("Failed to update post", "id", post.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update post"})
	}
	s.getPostCache.Reset()

	// Read again to return fully hydrated post
	updated, err := s.db.ReadPost(post.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error reading updated post"})
	}

	log.Info("Post updated", "id", idStr, "by", user.Username)
	return c.JSON(http.StatusOK, updated)
}
