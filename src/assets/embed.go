//go:build !dev

package assets

import (
	"embed"
	"io/fs"
)

//go:embed static
var embeddedAssets embed.FS

var Assets fs.FS = embeddedAssets
