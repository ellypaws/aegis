package bucket

import "context"

// Uploader provides a minimal interface for storing bytes in a bucket
// and returning a publicly accessible URL.
type Uploader interface {
	// Upload stores the given content under the provided key and returns a
	// public (or shareable) URL to access it.
	Upload(ctx context.Context, key string, body []byte, contentType string) (url string, err error)
	Key() string
}
