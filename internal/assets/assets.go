// Package assets embeds the application's static files (HTML templates,
// CSS, JavaScript, and vendored libraries like KaTeX) into the binary so
// hashcards can ship as a single self-contained executable.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static
var raw embed.FS

// FS is the embedded static tree, rooted so that paths like
// "css/tokens.css" or "templates/base.html" match the old on-disk
// static/ layout.
var FS fs.FS

func init() {
	sub, err := fs.Sub(raw, "static")
	if err != nil {
		// Only fails if the embed directive itself is wrong; that's a
		// build-time bug, not a runtime condition.
		panic(err)
	}
	FS = sub
}

// Sub returns the embedded subtree rooted at dir (e.g. "katex").
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
	return template.ParseFS(FS, "templates/base.html", "templates/"+page)
}
