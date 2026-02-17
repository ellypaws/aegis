package server

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	"github.com/labstack/echo/v4"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"drigo/pkg/exif"
	"drigo/pkg/flight"
	"drigo/pkg/types"

	"github.com/bwmarrin/discordgo"
)

var allowedWidths = map[int]bool{
	256: true, 500: true, 1000: true, 1080: true, 1920: true,
}

var allowedPercentages = map[int]bool{
	25: true, 50: true, 75: true,
}

const (
	cacheDir       = "cache"
	cacheTTL       = 24 * time.Hour
	defaultQuality = 90
	minQuality     = 1
	maxQuality     = 100
)

type resizeCacheEntry struct {
	Data        []byte
	ContentType string
}

// resizeFlightCache coalesces concurrent requests for the same cache key.
// The work function reads from the disk cache; if the entry is missing or
// stale it returns an error so the caller generates and persists.
var resizeFlightCache = flight.NewCache(func(key string) (resizeCacheEntry, error) {
	for _, ext := range []string{".webp", ".gif"} {
		diskPath := filepath.Join(cacheDir, key+ext)
		info, err := os.Stat(diskPath)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) >= cacheTTL {
			continue
		}
		data, err := os.ReadFile(diskPath)
		if err != nil {
			continue
		}
		ct := "image/webp"
		if ext == ".gif" {
			ct = "image/gif"
		}
		return resizeCacheEntry{Data: data, ContentType: ct}, nil
	}
	return resizeCacheEntry{}, fmt.Errorf("not in disk cache")
})

func init() {
	resizeFlightCache.Expiry(cacheTTL)
}

func (s *Server) handleGetResizedImage(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	wStr := c.QueryParam("w")
	pStr := c.QueryParam("p")
	if wStr == "" && pStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Query param 'w' or 'p' is required"})
	}

	var targetWidth int
	var isPercentage bool

	if wStr != "" {
		w, err := strconv.Atoi(wStr)
		if err != nil || !allowedWidths[w] {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid width; allowed: 256, 500, 1000, 1080, 1920"})
		}
		targetWidth = w
	} else {
		p, err := strconv.Atoi(pStr)
		if err != nil || !allowedPercentages[p] {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid percentage; allowed: 25, 50, 75"})
		}
		targetWidth = p
		isPercentage = true
	}

	quality := defaultQuality
	if qStr := c.QueryParam("q"); qStr != "" {
		q, err := strconv.Atoi(qStr)
		if err != nil || q < minQuality || q > maxQuality {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid quality; must be 1-100"})
		}
		quality = q
	}

	user := GetUserFromContext(c)
	post, err := s.db.GetPostByBlobID(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}
	if !canAccessPost(user, post) {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Access denied"})
	}

	blob, err := s.db.GetImageBlob(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	if isPercentage {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(blob.Data))
		if err != nil {
			log.Error("Failed to decode image config for percentage resize", "id", id, "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read image dimensions"})
		}
		targetWidth = cfg.Width * targetWidth / 100
		if targetWidth < 1 {
			targetWidth = 1
		}
	}

	cacheKey := fmt.Sprintf("%d_%d_q%d", id, targetWidth, quality)

	entry, err := resizeFlightCache.Get(cacheKey)
	if err == nil && len(entry.Data) > 0 {
		c.Response().Header().Set("Cache-Control", "private, max-age=86400")
		c.Response().Header().Set("Content-Type", entry.ContentType)
		c.Response().Header().Set("X-Cache", "hit")
		return c.Stream(http.StatusOK, entry.ContentType, bytes.NewReader(entry.Data))
	}

	isAnimatedGIF := false
	if ct := blob.GetContentType(); ct == "image/gif" {
		g, decErr := gif.DecodeAll(bytes.NewReader(blob.Data))
		if decErr == nil && len(g.Image) > 1 {
			isAnimatedGIF = true
		}
	}

	ext := ".webp"
	contentType := "image/webp"
	if isAnimatedGIF {
		ext = ".gif"
		contentType = "image/gif"
	}

	var result []byte
	if isAnimatedGIF {
		result, err = resizeAnimatedGIF(blob.Data, targetWidth)
	} else {
		result, err = resizeStatic(blob.Data, targetWidth, quality)
	}
	if err != nil {
		log.Error("Failed to resize image", "id", id, "width", targetWidth, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to resize image"})
	}

	diskPath := filepath.Join(cacheDir, cacheKey+ext)
	if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr != nil {
		log.Warn("Failed to create cache directory", "error", mkErr)
	} else if wErr := os.WriteFile(diskPath, result, 0o644); wErr != nil {
		log.Warn("Failed to write disk cache", "path", diskPath, "error", wErr)
	}

	go func() {
		_, _ = resizeFlightCache.Force(cacheKey)
	}()

	// Check if we need to inject EXIF for authenticated users
	// We only support PNG EXIF injection via pkg/exif.
	// We don't support GIF EXIF injection yet.
	if user != nil && contentType != "image/gif" {
		exifCacheKey := fmt.Sprintf("resize_exif_%s_%s", cacheKey, user.UserID)
		if cached, err := resizeExifCache.Get(exifCacheKey); err == nil {
			c.Response().Header().Set("Cache-Control", "private, max-age=86400")
			c.Response().Header().Set("Content-Type", "image/png")
			c.Response().Header().Set("Content-Disposition", "inline; filename=\"resized.png\"")
			c.Response().Header().Set("X-Cache", "hit-memory")
			return c.Stream(http.StatusOK, "image/png", bytes.NewReader(cached))
		}

		// Not in cache, convert WebP result to PNG with EXIF
		// Decode the WebP result we just got/generated
		img, _, err := image.Decode(bytes.NewReader(result))
		if err != nil {
			log.Error("Failed to decode resized image for EXIF injection", "error", err)
			// Fallback to sending original result
		} else {
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
			if err := encoder.Encode(&buf, img); err != nil {
				log.Error("Failed to encode resized PNG with EXIF", "error", err)
			} else {
				exifData := buf.Bytes()
				resizeExifCache.Set(exifCacheKey, exifData)

				c.Response().Header().Set("Cache-Control", "private, max-age=86400")
				c.Response().Header().Set("Content-Type", "image/png")
				c.Response().Header().Set("Content-Disposition", "inline; filename=\"resized.png\"")
				c.Response().Header().Set("X-Cache", "generated-memory")
				return c.Stream(http.StatusOK, "image/png", bytes.NewReader(exifData))
			}
		}
	}

	c.Response().Header().Set("Cache-Control", "private, max-age=86400")
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("X-Cache", "generated")
	return c.Stream(http.StatusOK, contentType, bytes.NewReader(result))
}

// resizeExifCache stores resized image data with EXIF metadata for specific users.
// Key: "resize_exif_{resizeCacheKey}_{userID}"
// Value: PNG bytes with EXIF
var resizeExifCache = flight.NewCache(func(key string) ([]byte, error) {
	return nil, fmt.Errorf("item not found")
})

func init() {
	resizeExifCache.Expiry(24 * time.Hour)
}

// resizeStatic decodes a static image, resizes it to the given width using
// Lanczos downsampling, and encodes the result as webp.
func resizeStatic(data []byte, width int, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	srcW := img.Bounds().Dx()

	if width >= srcW {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, webp.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("webp encode: %w", err)
		}
		return buf.Bytes(), nil
	}

	dst := imaging.Resize(img, width, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, dst, webp.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("webp encode: %w", err)
	}
	return buf.Bytes(), nil
}

// resizeAnimatedGIF composites frames respecting disposal methods, resizes each
// full-canvas composite using Lanczos, and re-encodes as an animated GIF.
func resizeAnimatedGIF(data []byte, width int) ([]byte, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gif decode: %w", err)
	}
	if len(g.Image) == 0 {
		return nil, fmt.Errorf("gif has no frames")
	}

	canvasW := g.Config.Width
	canvasH := g.Config.Height
	if canvasW == 0 || canvasH == 0 {
		canvasW = g.Image[0].Bounds().Dx()
		canvasH = g.Image[0].Bounds().Dy()
	}

	if width >= canvasW {
		return data, nil
	}

	ratio := float64(canvasH) / float64(canvasW)
	newH := int(float64(width) * ratio)
	if newH < 1 {
		newH = 1
	}

	canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	outFrames := make([]*image.Paletted, len(g.Image))
	outDisposal := make([]byte, len(g.Image))

	for i, frame := range g.Image {
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}

		// Snapshot for RestoreToPrevious before compositing.
		var snapshot *image.RGBA
		if disposal == gif.DisposalPrevious {
			snapshot = image.NewRGBA(canvas.Bounds())
			copy(snapshot.Pix, canvas.Pix)
		}

		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		resized := imaging.Resize(canvas, width, newH, imaging.Lanczos)

		paletted := image.NewPaletted(image.Rect(0, 0, width, newH), frame.Palette)
		draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), resized, image.Point{})

		outFrames[i] = paletted
		outDisposal[i] = gif.DisposalNone

		switch disposal {
		case gif.DisposalBackground:
			for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
				for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
					off := (y-canvas.Bounds().Min.Y)*canvas.Stride + (x-canvas.Bounds().Min.X)*4
					canvas.Pix[off+0] = 0
					canvas.Pix[off+1] = 0
					canvas.Pix[off+2] = 0
					canvas.Pix[off+3] = 0
				}
			}
		case gif.DisposalPrevious:
			if snapshot != nil {
				copy(canvas.Pix, snapshot.Pix)
			}
		}
	}

	g.Image = outFrames
	g.Disposal = outDisposal
	g.Config.Width = width
	g.Config.Height = newH

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, fmt.Errorf("gif encode: %w", err)
	}
	return buf.Bytes(), nil
}
