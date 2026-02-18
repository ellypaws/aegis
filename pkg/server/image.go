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
	"drigo/pkg/video"
)

func (s *Server) handleGetImage(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	settings, _ := s.db.GetSettings()
	publicAccess := settings != nil && settings.PublicAccess

	user := GetUserFromContext(c)
	if user == nil && !publicAccess {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Check permissions using the new helper
	post, err := s.db.GetPostByBlobID(uint(id))
	if err != nil {
		log.Error("Failed to find post for blob", "id", id, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	if !publicAccess && !canAccessPost(user, post) {
		log.Warn("Access denied for image", "blobID", id, "postID", post.ID, "user", user)
		if user != nil {
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
		}

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

	// Check cache first (only if we have a user to inject EXIF for)
	if user != nil {
		cacheKey := fmt.Sprintf("exif_%d_%s", id, user.UserID)
		if cached, err := imageExifCache.Get(cacheKey); err == nil {
			c.Response().Header().Set("Cache-Control", "private, max-age=86400")
			c.Response().Header().Set("Content-Type", "image/png")
			c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, downloadName))
			c.Response().Header().Set("X-Cache", "hit-memory")
			return c.Stream(http.StatusOK, "image/png", bytes.NewReader(cached))
		}
	}

	// Not in cache, generate if it's a PNG and we have a user
	if user != nil && contentType == "image/png" {
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

	// For non-PNGs (including videos), serve raw content directly
	c.Response().Header().Set("Cache-Control", "private, max-age=31536000")
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))
	return c.Stream(http.StatusOK, contentType, bytes.NewReader(blob.Data))
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

// videoPreviewFlightCache coalesces concurrent video preview requests and reads from disk cache.
var videoPreviewFlightCache = flight.NewCache(func(key string) ([]byte, error) {
	diskPath := filepath.Join(cacheDir, key+".gif")
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
	// Initialize with longer TTL for videos as they are heavier to generate
	videoPreviewFlightCache.Expiry(24 * time.Hour)
}

func (s *Server) handleGetThumb(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	// 1. Try to get existing thumbnail from DB
	thumb, err := s.db.GetImageThumbnailByBlobID(uint(id))
	if err == nil && len(thumb) > 0 {
		// Detect content type of thumbnail
		contentType := http.DetectContentType(thumb)
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
		c.Response().Header().Set("Content-Type", contentType)
		// Trigger preload for full image EXIF if authorized (and if it's an image)
		// We skip this for now or check if it's an image.
		return c.Stream(http.StatusOK, contentType, bytes.NewReader(thumb))
	}

	// 2. No thumbnail, fetch blob to check type and generate fallback
	blob, err := s.db.GetImageBlob(uint(id))
	if err != nil {
		// Fallback to blur if no thumbnail and can't get blob?
		// Actually if we can't get blob we can't do anything.
		return s.handleGetBlur(c)
	}

	// Helper to check for video
	isVideo := func(contentType, filename string) bool {
		if strings.HasPrefix(contentType, "video/") {
			return true
		}
		ext := strings.ToLower(filepath.Ext(filename))
		return ext == ".mp4" || ext == ".webm" || ext == ".mov" || ext == ".mkv"
	}

	if isVideo(blob.GetContentType(), blob.Filename) {
		return s.serveVideoPreview(c, uint(id), blob)
	}

	// 3. Fallback for images: use existing handleGetBlur logic via function call
	// We can't easily reuse handleGetBlur 'as is' because it reparses ID and refetches blob.
	// But mostly it's fine to just call it if we want to be lazy, or refactor logic.
	// Since we already have the blob, let's call a helper or just do the blur here?
	// The original handleGetThumb called s.handleGetBlur(c).
	// Let's just do that for now to minimize code duplication, although it will double-fetch blob.
	// Optimization: We could refactor handleGetBlur to take a blob, but for now this is safe.
	return s.handleGetBlur(c)
}

func (s *Server) serveVideoPreview(c echo.Context, id uint, blob *types.ImageBlob) error {
	user := GetUserFromContext(c)
	settings, _ := s.db.GetSettings()
	publicAccess := settings != nil && settings.PublicAccess

	// Check authorization
	isAuthorized := publicAccess
	if !isAuthorized && user != nil {
		post, err := s.db.GetPostByBlobID(id)
		if err == nil && canAccessPost(user, post) {
			isAuthorized = true
		}
	}

	// Cache key differentiation
	suffix := "auth"
	if !isAuthorized {
		suffix = "blur"
	}
	cacheKey := fmt.Sprintf("vid_prev_%d_%s", id, suffix)

	// Check cache
	data, err := videoPreviewFlightCache.Get(cacheKey)
	if err == nil && len(data) > 0 {
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
		c.Response().Header().Set("Content-Type", "image/gif")
		c.Response().Header().Set("X-Cache", "hit")
		return c.Stream(http.StatusOK, "image/gif", bytes.NewReader(data))
	}

	// Generate
	// For authorized: 2fps
	// For unauthorized: 1fps + blur
	fps := 2
	blurry := false
	if !isAuthorized {
		fps = 1
		blurry = true
	}

	log.Info("Generating video preview", "id", id, "authorized", isAuthorized, "blurry", blurry)

	gifData, err := video.GeneratePreviewGIF(blob.Data, fps, blurry)
	if err != nil {
		log.Error("Failed to generate video preview", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate preview"})
	}

	// Save to disk cache
	diskPath := filepath.Join(cacheDir, cacheKey+".gif")
	if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr != nil {
		log.Warn("Failed to create cache directory", "error", mkErr)
	} else if wErr := os.WriteFile(diskPath, gifData, 0o644); wErr != nil {
		log.Warn("Failed to write video preview cache", "path", diskPath, "error", wErr)
	}

	// Update flight cache
	go func() {
		_, _ = videoPreviewFlightCache.Force(cacheKey)
	}()

	c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
	c.Response().Header().Set("Content-Type", "image/gif")
	c.Response().Header().Set("X-Cache", "generated")
	return c.Stream(http.StatusOK, "image/gif", bytes.NewReader(gifData))
}
