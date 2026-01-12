//go:build dev
// +build dev

package dabbi

import (
	"io/fs"
	"os"
)

// GetUIFS returns the UI filesystem from disk for development
func GetUIFS() (fs.FS, error) {
	return os.DirFS("ui/dist"), nil
}
