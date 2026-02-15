package app

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embeddedFiles embed.FS

// FS returns the embedded filesystem for the app
func FS() fs.FS {
	fsys, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		panic(err)
	}
	return fsys
}
