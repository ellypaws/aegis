package server

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/gen2brain/webp"
	"github.com/labstack/echo/v4"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

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

	c.Response().Header().Set("Cache-Control", "private, max-age=31536000") // Private cache since it's auth-dependent
	c.Response().Header().Set("Content-Type", contentType)

	return c.Stream(http.StatusOK, contentType, bytes.NewReader(blob.Data))
}

func (s *Server) handleGetBlur(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	// Blur is public
	blob, err := s.db.GetImageBlob(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	// Generate blur
	img, _, err := image.Decode(bytes.NewReader(blob.Data))
	if err != nil {
		log.Error("Failed to decode image for blur", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode image"})
	}

	// Resize to very small (e.g. 16px wide)
	bounds := img.Bounds()
	ratio := float64(bounds.Dx()) / float64(bounds.Dy())
	w := 16
	h := int(float64(w) / ratio)
	if h < 1 {
		h = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, dst, webp.Options{Quality: 40}); err != nil {
		log.Error("Failed to encode blur webp", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode blur"})
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
	c.Response().Header().Set("Content-Type", "image/webp")
	return c.Stream(http.StatusOK, "image/webp", &buf)
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
