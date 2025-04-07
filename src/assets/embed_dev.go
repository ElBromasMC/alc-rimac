//go:build dev

package assets

import (
	"io/fs"
	"os"
)

var Assets fs.FS

func init() {
	Assets = os.DirFS("assets")
}
