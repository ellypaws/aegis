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
	for _, rid := range claims.RoleIDs {
		userRoleMap[rid] = true
	}

	for _, ar := range p.AllowedRoles {
		if userRoleMap[ar.RoleID] {
			return true
		}
	}

	return false
}
