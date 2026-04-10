package media

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/asano69/hashcards/internal/errs"
)

// LoadResult holds the raw bytes and derived metadata of a loaded media file.
type LoadResult struct {
	// Data is the raw file content.
	Data []byte
	// MimeType is the MIME type of the file (e.g. "image/png").
	MimeType string
}

// DataURL returns a base64-encoded data URI suitable for use in an HTML
// <img src="..."> attribute. This lets the drill server serve cards without
// separate HTTP requests for each image.
func (r LoadResult) DataURL() string {
	encoded := base64.StdEncoding.EncodeToString(r.Data)
	return fmt.Sprintf("data:%s;base64,%s", r.MimeType, encoded)
}

// Load reads a media file from disk given a ResolveResult. It returns an
// error if the file cannot be read or does not exist.
func Load(resolved ResolveResult) (LoadResult, error) {
	data, err := os.ReadFile(resolved.AbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return LoadResult{}, errs.Newf("media file not found: %s", resolved.AbsPath)
		}
		return LoadResult{}, errs.Newf("read media file %s: %v", resolved.AbsPath, err)
	}
	return LoadResult{Data: data, MimeType: resolved.MimeType}, nil
}

// LoadFromPath resolves and loads a media file in one step. It is a
// convenience wrapper around Resolve followed by Load.
func LoadFromPath(deckFilePath, ref string) (LoadResult, error) {
	resolved, err := Resolve(deckFilePath, ref)
	if err != nil {
		return LoadResult{}, err
	}
	return Load(resolved)
}
