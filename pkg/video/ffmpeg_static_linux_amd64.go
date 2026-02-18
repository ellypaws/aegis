//go:build ffstatic && linux && amd64

package video

import (
	ffstatic "github.com/go-ffstatic/linux-amd64"
)

func init() {
	setupStaticFFmpeg(ffstatic.FFmpegPath(), ffstatic.FFprobePath(), "ffmpeg", "ffprobe")
}
