//go:build ffstatic && windows && amd64

package video

import (
	ffstatic "github.com/go-ffstatic/windows-amd64"
)

func init() {
	setupStaticFFmpeg(ffstatic.FFmpegPath(), ffstatic.FFprobePath(), "ffmpeg.exe", "ffprobe.exe")
}
