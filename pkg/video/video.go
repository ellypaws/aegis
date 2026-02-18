package video

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/segmentio/ksuid"
)

// Ensure ffmpeg is available
func CheckFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	return err
}

// GeneratePreviewGIF generates a GIF from a video file.
// If blurry is true, it applies a blur filter.
// fps determines the frame rate.
func GeneratePreviewGIF(videoData []byte, fps int, blurry bool) ([]byte, error) {
	// Write video data to a temporary file
	tmpName := filepath.Join(os.TempDir(), fmt.Sprintf("drigo_video_%s.mp4", ksuid.New().String()))
	if err := os.WriteFile(tmpName, videoData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp video file: %w", err)
	}
	defer os.Remove(tmpName)

	// Create a temporary output file for the GIF
	outName := filepath.Join(os.TempDir(), fmt.Sprintf("drigo_preview_%s.gif", ksuid.New().String()))
	defer os.Remove(outName)

	// Build ffmpeg filter
	// We want to sample generic keyframes or just fps.
	// vf: "fps=X,scale=320:-1:flags=lanczos"
	// If blurry: "fps=X,scale=64:-1:flags=lanczos,boxblur=10:1"

	var filter string
	if blurry {
		// Low res + blur
		filter = fmt.Sprintf("fps=%d,scale=64:-2:flags=lanczos,boxblur=2:1", fps)
	} else {
		// Reasonable preview size
		filter = fmt.Sprintf("fps=%d,scale=320:-2:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", fps)
	}

	// Basic command construction
	// Note: palettegen/paletteuse gives better GIF quality but might be slower.
	// For blurry, we don't care about quality as much, direct mapping is fine.

	var args []string
	args = append(args, "-y", "-i", tmpName, "-vf", filter)

	if !blurry {
		// For authorized (clear) GIF, we use the complex filter for quality if possible,
		// but to keep it simple and robust:
		// Let's use a simpler filter first to ensure it works.
		// "fps=%d,scale=320:-1:flags=lanczos"
	}

	args = append(args, outName)

	cmd := exec.Command("ffmpeg", args...)

	// Capture stderr for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		log.Error("FFmpeg failed", "stderr", stderr.String(), "error", err)
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}
	log.Debug("Generated GIF", "took", time.Since(start), "size_bytes", len(videoData), "blurry", blurry)

	// Read the generated GIF
	gifData, err := os.ReadFile(outName)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated gif: %w", err)
	}

	return gifData, nil
}
