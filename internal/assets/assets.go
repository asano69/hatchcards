package assets

import (
	"embed"
	"io/fs"
)

//go:embed static/dist
var rawStatic embed.FS

var FS fs.FS

func init() {
	sub, err := fs.Sub(rawStatic, "static/dist")
	if err != nil {
		panic(err)
	}
	FS = sub
}
