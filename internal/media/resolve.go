// Package media handles discovery, validation, and loading of media files
// (images and audio) referenced from deck Markdown files.
package media

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/asano69/hashcards/internal/errs"
)

// SupportedExtensions is the set of media file extensions the application accepts.
// This includes both image and audio formats, matching the Rust implementation.
var SupportedExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".webp": {},
	".svg":  {},
	".mp3":  {},
	".wav":  {},
	".ogg":  {},
	".m4a":  {},
}

// ResolveResult holds a resolved media path and its MIME type.
type ResolveResult struct {
	// AbsPath is the absolute path to the media file on disk.
	AbsPath string
	// MimeType is derived from the file extension (e.g. "image/png").
	MimeType string
}

// Resolve resolves a media reference (as it appears in a Markdown image tag)
// to an absolute path.
//
// ref may be:
//   - a collection-root-relative path prefixed with "@/" (e.g. "@/thetempest.webp")
//   - a relative path ("images/cat.png")
//   - a path with leading "./" ("./cat.png")
//
// For "@/" paths, deckFilePath is used only to find the collection root (the
// directory passed to the check/drill command). In practice the Go port passes
// the deck file's absolute path, so the collection root is its parent.
//
// Absolute paths and URLs (http://, https://, data:) are rejected.
func Resolve(deckFilePath, ref string) (ResolveResult, error) {
	if isURL(ref) {
		return ResolveResult{}, errs.Newf("external URLs are not supported as media references: %s", ref)
	}
	if filepath.IsAbs(ref) {
		return ResolveResult{}, errs.Newf("absolute media paths are not allowed: %s", ref)
	}

	deckDir := filepath.Dir(deckFilePath)

	var absPath string
	if strings.HasPrefix(ref, "@/") {
		// Collection-root-relative path: resolve relative to the collection
		// root directory (the parent of the deck file's directory).
		// When the deck file sits directly in the collection root, deckDir IS
		// the collection root, which is the common case.
		stripped := ref[2:] // remove "@/"
		absPath = filepath.Join(deckDir, filepath.FromSlash(stripped))
	} else {
		// Deck-relative path.
		absPath = filepath.Join(deckDir, filepath.FromSlash(ref))
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	if _, ok := SupportedExtensions[ext]; !ok {
		return ResolveResult{}, errs.Newf("unsupported media file extension %q in %s", ext, ref)
	}

	mime, err := mimeForExt(ext)
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{AbsPath: absPath, MimeType: mime}, nil
}

// FindMediaFiles returns the absolute paths of all media files found directly
// inside the directory that contains deckFilePath. Subdirectories are not
// traversed; the caller is responsible for recursive discovery if needed.
func FindMediaFiles(deckFilePath string) ([]string, error) {
	deckDir := filepath.Dir(deckFilePath)
	entries, err := os.ReadDir(deckDir)
	if err != nil {
		return nil, errs.Newf("read deck directory %s: %v", deckDir, err)
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := SupportedExtensions[ext]; ok {
			paths = append(paths, filepath.Join(deckDir, entry.Name()))
		}
	}
	return paths, nil
}

// isURL reports whether ref is an external URL or data URI.
func isURL(ref string) bool {
	return strings.HasPrefix(ref, "http://") ||
		strings.HasPrefix(ref, "https://") ||
		strings.HasPrefix(ref, "data:")
}

// mimeForExt maps a lowercase file extension to its MIME type.
func mimeForExt(ext string) (string, error) {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	case ".svg":
		return "image/svg+xml", nil
	case ".mp3":
		return "audio/mpeg", nil
	case ".wav":
		return "audio/wav", nil
	case ".ogg":
		return "audio/ogg", nil
	case ".m4a":
		return "audio/mp4", nil
	default:
		return "", errs.Newf("no MIME type for extension %q", ext)
	}
}
