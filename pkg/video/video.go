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
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var ffmpegBinary = "ffmpeg"

// CheckFFmpeg ensures ffmpeg is available
func CheckFFmpeg() error {
	_, err := exec.LookPath(ffmpegBinary)
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

	var filter string
	if blurry {
		// Low res + blur
		filter = fmt.Sprintf("fps=%d,scale=64:-2:flags=lanczos,boxblur=2:1", fps)
	} else {
		// Reasonable preview size
		filter = fmt.Sprintf("fps=%d,scale=320:-2:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", fps)
	}

	var args []string
	args = append(args, "-y", "-i", tmpName, "-vf", filter)

	args = append(args, outName)

	cmd := exec.Command(ffmpegBinary, args...)

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

// ResizeToWebM reshapes a video or animated GIF to WebM with the given width,
// preserving the original frame rate.
func ResizeToWebM(data []byte, width int) ([]byte, error) {
	// Write input data to a temp file
	tmpName := filepath.Join(os.TempDir(), fmt.Sprintf("drigo_input_%s", ksuid.New().String()))
	if err := os.WriteFile(tmpName, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp input file: %w", err)
	}
	defer os.Remove(tmpName)

	// Create a temporary output file
	outName := filepath.Join(os.TempDir(), fmt.Sprintf("drigo_output_%s.webm", ksuid.New().String()))
	// Defer removal is good practice, but we read it right after.
	defer os.Remove(outName)

	err := ffmpeg.Input(tmpName).
		Output(outName, ffmpeg.KwArgs{
			"vf":       fmt.Sprintf("scale=%d:-2:flags=lanczos", width),
			"c:v":      "libvpx-vp9",
			"b:v":      "0",
			"crf":      "30",
			"an":       "",  // No audio
			"cpu-used": "2", // Speed/Quality balance
		}).
		OverWriteOutput().
		ErrorToStdOut().
		Run()

	if err != nil {
		return nil, fmt.Errorf("ffmpeg-go run failed: %w", err)
	}

	// Read the generated WebM
	webmData, err := os.ReadFile(outName)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated webm: %w", err)
	}

	return webmData, nil
}
