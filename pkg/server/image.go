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

	// Check permissions using the new helper
	post, err := s.db.GetPostByBlobID(uint(id))
	if err != nil {
		log.Error("Failed to find post for blob", "id", id, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	if !canAccessPost(user, post) {
		log.Warn("Access denied for image", "blobID", id, "postID", post.ID, "user", user != nil)
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
		} else {
			log.Info("Debug Access", "user", "nil", "postAllowedCount", len(post.AllowedRoles))
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

	// If not authenticated or not a PNG/JPEG that we want to convert to PNG with EXIF, just serve raw.
	// Actually, only PNG supports this specific EXIF insertion via pkg/exif.
	// So if it's not a PNG, we might skip EXIF or we have to convert it.
	// The requirement says "Add discord user exif data ... make sure these are both cached only in memory".
	// We'll stick to PNG for now as pkg/exif is for PNG.
	isPNG := blob.GetContentType() == "image/png"
	// We also support JPEG converting to PNG for EXIF if we really wanted to, but let's stick to PNG for now unless requested.
	// If the user uploads a JPEG, we could decode and re-encode as PNG with EXIF.
	// For now, let's assume we do this for all supported image types if we can easily convert,
	// or just for PNG if we want to be safe.
	// The prompt implies we should add it, so let's try to do it for all images by converting to PNG if needed,
	// OR just for PNGs.
	// Given "pkg/exif referencing pkg/drigo/handlers.go on how I built MemberExif", looking at handlers.go it encodes to PNG.
	// "if imgBlob.ContentType == "image/png" { ... encoder.EncodeReader ... } else { ... }"
	// So in handlers.go it seems it only encodes EXIF for PNGs?
	// Wait, in handlers.go:
	// if imgBlob.ContentType == "image/png" { ... } else { images = append(images, bytes.NewReader(imgBlob.Data)) }
	// So it ONLY adds EXIF for PNGs. We will follow that pattern.

	if user != nil && isPNG {
		cacheKey := fmt.Sprintf("exif_%d_%s", id, user.UserID)
		// Force filename to have .png extension since we are serving a PNG
		downloadName := filename
		if !strings.HasSuffix(strings.ToLower(downloadName), ".png") {
			ext := filepath.Ext(downloadName)
			if ext != "" {
				downloadName = strings.TrimSuffix(downloadName, ext) + ".png"
			} else {
				downloadName += ".png"
			}
		}

		if cached, err := imageExifCache.Get(cacheKey); err == nil {
			c.Response().Header().Set("Cache-Control", "private, max-age=86400")
			c.Response().Header().Set("Content-Type", "image/png")
			c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, downloadName))
			c.Response().Header().Set("X-Cache", "hit-memory")
			return c.Stream(http.StatusOK, "image/png", bytes.NewReader(cached))
		}

		// Not in cache, generate
		member := &discordgo.Member{
			User: &discordgo.User{
				ID:            user.UserID,
				Username:      user.Username,
				Discriminator: "0",
				GlobalName:    user.Username,
			},
			Nick: user.Username, // Best fallback since we don't have per-guild nickname in claims
		}

		memberExif := types.ToMemberExif(member)
		// Fix up because ToMemberExif uses member.User which we constructed, but also member.DisplayName()
		// which checks Nick then User.GlobalName then User.Username.
		// We set Nick to user.DisplayName() which is checking GlobalName/Username.
		// So this should be fine.

		encoder := exif.NewEncoder(memberExif)
		var buf bytes.Buffer
		if err := encoder.EncodeReader(&buf, bytes.NewReader(blob.Data)); err != nil {
			log.Error("Failed to encode EXIF", "error", err)
			return c.Stream(http.StatusOK, contentType, bytes.NewReader(blob.Data))
		}

		exifData := buf.Bytes()
		imageExifCache.Set(cacheKey, exifData)

		c.Response().Header().Set("Cache-Control", "private, max-age=86400")
		c.Response().Header().Set("Content-Type", "image/png")
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, downloadName))
		c.Response().Header().Set("X-Cache", "generated-memory")
		return c.Stream(http.StatusOK, "image/png", bytes.NewReader(exifData))
	}

	c.Response().Header().Set("Cache-Control", "private, max-age=31536000") // Private cache since it's auth-dependent
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))

	return c.Stream(http.StatusOK, contentType, bytes.NewReader(blob.Data))
}

// imageExifCache stores image data with EXIF metadata for specific users.
// Key: "exif_{blobID}_{userID}"
// Value: PNG bytes with EXIF
var imageExifCache = flight.NewCache(func(key string) ([]byte, error) {
	// We cannot generate the value here because we need the blob and user data,
	// which are not available in this scope.
	// This cache is populated manually via Set().
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

	// Thumbnails are generally considered public if the blur is public,
	// BUT for stricter security we could check permissions.
	// The user request says "serve the thumbnail unblurred if it's available for unauthorized/logged out users".
	// So we should NOT enforce strict auth on the thumbnail, or at least be lenient.
	// However, usually we might want to check if the user *could* see the post if they were logged in?
	// But the user explicitly said: "We should always be serving the thumbnail unblurred if it's available for unauthorized/logged out users."
	// So effectively, thumbnails are public.

	thumb, err := s.db.GetImageThumbnailByBlobID(uint(id))
	if err != nil || len(thumb) == 0 {
		// Fallback to blur if no thumbnail
		return s.handleGetBlur(c)
	}

	// Detect content type of thumbnail
	contentType := http.DetectContentType(thumb)
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
	c.Response().Header().Set("Content-Type", contentType)

	return c.Stream(http.StatusOK, contentType, bytes.NewReader(thumb))
}
