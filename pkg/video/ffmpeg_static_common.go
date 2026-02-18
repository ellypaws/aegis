//go:build ffstatic

package video

import (
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func setupStaticFFmpeg(ffmpegPath, ffprobePath, ffmpegName, ffprobeName string) {
	if ffmpegPath == "" || ffprobePath == "" {
		log.Error("ffstatic returned empty paths", "ffmpeg", ffmpegPath, "ffprobe", ffprobePath)
		return
	}

	// Update the internal binary path for direct exec usage
	ffmpegBinary = ffmpegPath

	// Create a shim directory to prepend to PATH
	// This ensures exec.LookPath("ffmpeg") finds our static binary
	myDir := filepath.Join(os.TempDir(), "aegis_ffmpeg")
	if err := os.MkdirAll(myDir, 0755); err != nil {
		log.Error("Failed to create temp dir for ffmpeg shim", "err", err)
		return
	}

	// Helper to copy/link file
	linkOrCopy := func(src, dstName string) {
		dst := filepath.Join(myDir, dstName)
		// Remove if exists
		os.Remove(dst)

		// Try Symlink first (fastest, but might fail on Windows without privileges)
		if err := os.Symlink(src, dst); err == nil {
			return
		}

		// Try Hardlink (fast, but must be same drive)
		if err := os.Link(src, dst); err == nil {
			return
		}

		// Fallback to Copy
		in, err := os.Open(src)
		if err != nil {
			log.Error("Failed to open static binary for copying", "src", src, "err", err)
			return
		}
		defer in.Close()

		out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Error("Failed to create shim file", "dst", dst, "err", err)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			log.Error("Failed to copy static binary", "err", err)
		}
	}

	linkOrCopy(ffmpegPath, ffmpegName)
	linkOrCopy(ffprobePath, ffprobeName)

	// Update PATH
	path := os.Getenv("PATH")
	os.Setenv("PATH", myDir+string(os.PathListSeparator)+path)

	log.Debug("Initialized static ffmpeg", "path", ffmpegPath, "shim_dir", myDir)
}
