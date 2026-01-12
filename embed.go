//go:build !dev
// +build !dev

package dabbi

import (
	"embed"
	"io/fs"
)

//go:embed ui/dist/*
var embeddedUI embed.FS

// GetUIFS returns the embedded UI filesystem
func GetUIFS() (fs.FS, error) {
	return fs.Sub(embeddedUI, "ui/dist")
}
