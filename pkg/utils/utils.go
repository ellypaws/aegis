package utils

import (
	"net/http"
	"strings"
)

// Deprecated: Use new(T) from 1.26 syntax.
func Pointer[T any](a T) *T {
	return &a
}

func ContentType(data []byte) string {
	if len(data) >= 512 {
		data = data[:512]
	}
	return http.DetectContentType(data)
}

func IsVideoType(data []byte) bool {
	return strings.HasPrefix(ContentType(data), "video/")
}

func IsVideoContentType(ct string) bool {
	return strings.HasPrefix(ct, "video/")
}

func GetFileExtension(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/svg+xml":
		return "svg"
	case "video/mp4":
		return "mp4"
	case "video/quicktime", "video/mov":
		return "mov"
	case "video/webm":
		return "webm"
	case "video/ogg":
		return "ogv"
	default:
		return "bin"
	}
}
