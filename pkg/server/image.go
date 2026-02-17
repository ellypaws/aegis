package server

import (
	"bytes"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	"github.com/labstack/echo/v4"
	_ "golang.org/x/image/webp"

	"drigo/pkg/exif"
	"drigo/pkg/flight"
	"drigo/pkg/types"
)

func (s *Server) handleGetImage(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	user := GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Check permissions using the new helper
	post, err := s.db.GetPostByBlobID(uint(id))
	if err != nil {
		log.Error("Failed to find post for blob", "id", id, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	if !canAccessPost(user, post) {
		log.Warn("Access denied for image", "blobID", id, "postID", post.ID, "user", user)
		roleIDs := make([]string, 0, len(user.Roles))
		for _, r := range user.Roles {
			if r != nil {
				roleIDs = append(roleIDs, r.ID)
			}
		}

		allowedIDs := make([]string, 0, len(post.AllowedRoles))
		for _, ar := range post.AllowedRoles {
			allowedIDs = append(allowedIDs, ar.RoleID)
		}
		log.Info("Debug Access", "userRoles", roleIDs, "allowedRoles", allowedIDs, "isAdmin", user.IsAdmin)

		return c.JSON(http.StatusForbidden, map[string]string{"error": "Access denied"})
	}

	blob, err := s.db.GetImageBlob(uint(id))
	if err != nil {
		log.Error("Failed to get image blob", "id", id, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	contentType := blob.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := blob.Filename
	if filename == "" {
		ext := blob.GetFileExtension()
		title := post.Title
		if title == "" {
			title = "image"
		}
		filename = fmt.Sprintf("%s.%s", title, ext)
	}

	disposition := "inline"
	if c.QueryParam("download") == "1" {
		disposition = "attachment"
	}

	// Force filename to have .png extension since we are serving a PNG for EXIF
	downloadName := filename
	if !strings.HasSuffix(strings.ToLower(downloadName), ".png") {
		ext := filepath.Ext(downloadName)
		if ext != "" {
			downloadName = strings.TrimSuffix(downloadName, ext) + ".png"
		} else {
			downloadName += ".png"
		}
	}

	// Check cache first
	cacheKey := fmt.Sprintf("exif_%d_%s", id, user.UserID)
	if cached, err := imageExifCache.Get(cacheKey); err == nil {
		c.Response().Header().Set("Cache-Control", "private, max-age=86400")
		c.Response().Header().Set("Content-Type", "image/png")
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, downloadName))
		c.Response().Header().Set("X-Cache", "hit-memory")
		return c.Stream(http.StatusOK, "image/png", bytes.NewReader(cached))
	}

	// Not in cache, generate
	exifData, err := s.generateAndCacheImageExif(uint(id), user)
	if err != nil {
		log.Error("Failed to generate EXIF", "error", err)
		// Serve raw image with original headers on failure
		c.Response().Header().Set("Cache-Control", "private, max-age=31536000")
		c.Response().Header().Set("Content-Type", contentType)
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))
		return c.Stream(http.StatusOK, contentType, bytes.NewReader(blob.Data))
	}

	c.Response().Header().Set("Cache-Control", "private, max-age=86400")
	c.Response().Header().Set("Content-Type", "image/png")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, downloadName))
	c.Response().Header().Set("X-Cache", "generated-memory")
	return c.Stream(http.StatusOK, "image/png", bytes.NewReader(exifData))
}

func (s *Server) isImageExifCached(id uint, user *JwtCustomClaims) bool {
	if user == nil {
		return false
	}
	cacheKey := fmt.Sprintf("exif_%d_%s", id, user.UserID)
	_, err := imageExifCache.Get(cacheKey)
	return err == nil
}

func (s *Server) generateAndCacheImageExif(id uint, user *JwtCustomClaims) ([]byte, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	cacheKey := fmt.Sprintf("exif_%d_%s", id, user.UserID)
	// Check cache again just in case (though caller might have checked)
	if cached, err := imageExifCache.Get(cacheKey); err == nil {
		return cached, nil
	}

	blob, err := s.db.GetImageBlob(id)
	if err != nil {
		return nil, err
	}

	if blob.GetContentType() != "image/png" {
		// Currently only supporting PNG for EXIF injection pattern
		return nil, fmt.Errorf("not a png")
	}

	member := &discordgo.Member{
		User: &discordgo.User{
			ID:            user.UserID,
			Username:      user.Username,
			Discriminator: "0",
			GlobalName:    user.Username,
		},
		Nick: user.Username,
	}

	memberExif := types.ToMemberExif(member)
	encoder := exif.NewEncoder(memberExif)
	var buf bytes.Buffer
	if err := encoder.EncodeReader(&buf, bytes.NewReader(blob.Data)); err != nil {
		return nil, err
	}

	exifData := buf.Bytes()
	imageExifCache.Set(cacheKey, exifData)
	return exifData, nil
}

var imageExifCache = flight.NewCache(func(key string) ([]byte, error) {
	return nil, fmt.Errorf("item not found")
})

func init() {
	imageExifCache.Expiry(24 * time.Hour)
}

// blurFlightCache coalesces concurrent blur requests and reads from disk cache.
var blurFlightCache = flight.NewCache(func(key string) ([]byte, error) {
	diskPath := filepath.Join(cacheDir, key+".webp")
	info, err := os.Stat(diskPath)
	if err != nil {
		return nil, fmt.Errorf("not in disk cache")
	}
	if time.Since(info.ModTime()) >= cacheTTL {
		return nil, fmt.Errorf("stale")
	}
	return os.ReadFile(diskPath)
})

func init() {
	blurFlightCache.Expiry(cacheTTL)
}

func (s *Server) handleGetBlur(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	cacheKey := fmt.Sprintf("blur_%d", id)

	data, err := blurFlightCache.Get(cacheKey)
	if err == nil && len(data) > 0 {
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
		c.Response().Header().Set("Content-Type", "image/webp")
		c.Response().Header().Set("X-Cache", "hit")
		return c.Stream(http.StatusOK, "image/webp", bytes.NewReader(data))
	}

	blob, err := s.db.GetImageBlob(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	img, err := imaging.Decode(bytes.NewReader(blob.Data))
	if err != nil {
		log.Error("Failed to decode image for blur", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode image"})
	}

	img = imaging.Resize(img, 64, 0, imaging.Lanczos)
	img = imaging.Blur(img, 8)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: 40}); err != nil {
		log.Error("Failed to encode blur webp", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode blur"})
	}

	result := buf.Bytes()

	diskPath := filepath.Join(cacheDir, cacheKey+".webp")
	if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr != nil {
		log.Warn("Failed to create cache directory", "error", mkErr)
	} else if wErr := os.WriteFile(diskPath, result, 0o644); wErr != nil {
		log.Warn("Failed to write blur disk cache", "path", diskPath, "error", wErr)
	}

	go func() {
		_, _ = blurFlightCache.Force(cacheKey)
	}()

	c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
	c.Response().Header().Set("Content-Type", "image/webp")
	c.Response().Header().Set("X-Cache", "generated")
	return c.Stream(http.StatusOK, "image/webp", bytes.NewReader(result))
}

func canAccessPost(claims *JwtCustomClaims, p *types.Post) bool {
	if claims != nil && claims.IsAdmin {
		return true
	}
	// If no roles required, public
	if len(p.AllowedRoles) == 0 {
		return true
	}
	if claims == nil {
		return false
	}

	// Check intersection
	userRoleMap := make(map[string]bool)
	for _, r := range claims.Roles {
		if r != nil {
			userRoleMap[r.ID] = true
		}
	}

	for _, ar := range p.AllowedRoles {
		if userRoleMap[ar.RoleID] {
			return true
		}
	}

	return false
}

func (s *Server) handleGetThumb(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	thumb, err := s.db.GetImageThumbnailByBlobID(uint(id))
	if err != nil || len(thumb) == 0 {
		// Fallback to blur if no thumbnail
		return s.handleGetBlur(c)
	}

	// Trigger preload for full image EXIF
	user := GetUserFromContext(c)
	if user != nil {
		s.preloadQueue.Enqueue(uint(id), user)
	}

	// Detect content type of thumbnail
	contentType := http.DetectContentType(thumb)
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
	c.Response().Header().Set("Content-Type", contentType)

	return c.Stream(http.StatusOK, contentType, bytes.NewReader(thumb))
}
