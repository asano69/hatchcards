// internal/assets/assets.go
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static
var rawStatic embed.FS

//go:embed templates
var rawTemplates embed.FS

// FS is the embedded static tree (CSS, JS, KaTeX, favicon), served
// publicly under /static/.
var FS fs.FS

// templatesFS is the embedded HTML template tree. Kept separate from FS
// so templates are never exposed over HTTP.
var templatesFS fs.FS

func init() {
	sub, err := fs.Sub(rawStatic, "static")
	if err != nil {
		// Only fails if the embed directive itself is wrong; that's a
		// build-time bug, not a runtime condition.
		panic(err)
	}
	FS = sub

	tsub, err := fs.Sub(rawTemplates, "templates")
	if err != nil {
		panic(err)
	}
	templatesFS = tsub
}

// Sub returns the embedded static subtree rooted at dir (e.g. "katex").
func Sub(dir string) fs.FS {
	sub, err := fs.Sub(FS, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// ParseTemplate parses base.html together with the named page template
// from the embedded templates/ directory.
func ParseTemplate(page string) (*template.Template, error) {
	return template.ParseFS(templatesFS, "base.html", page)
}
