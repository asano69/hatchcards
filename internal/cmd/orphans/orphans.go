// Package orphans implements the "orphans" subcommand, which lists media files
// that exist on disk but are not referenced by any card in the collection.
package orphans

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/media"
	"github.com/asano69/hashcards/internal/types"
)

// Run loads the collection at root and writes the absolute path of every
// media file that is not referenced by any card, one path per line.
func Run(root string, out io.Writer) error {
	database, err := db.Open(":memory:")
	if err != nil {
		return err
	}
	defer database.Close()

	col, err := collection.Load(root, database)
	if err != nil {
		return err
	}

	// Collect every media file referenced by any card.
	referenced := make(map[string]struct{})
	for _, card := range col.Cards {
		for _, ref := range cardImageRefs(card) {
			resolved, err := media.Resolve(card.FilePath(), ref)
			if err != nil {
				// Unresolvable references are reported by "check"; skip here.
				continue
			}
			referenced[resolved.AbsPath] = struct{}{}
		}
	}

	// Collect every media file that exists alongside any deck file.
	presentFiles := make(map[string]struct{})
	for _, card := range col.Cards {
		paths, err := media.FindMediaFiles(card.FilePath())
		if err != nil {
			return err
		}
		for _, p := range paths {
			presentFiles[p] = struct{}{}
		}
	}

	// Orphans are present on disk but not referenced by any card.
	var orphans []string
	for path := range presentFiles {
		if _, ok := referenced[path]; !ok {
			orphans = append(orphans, path)
		}
	}
	sort.Strings(orphans)

	for _, path := range orphans {
		fmt.Fprintln(out, path)
	}

	return nil
}

// cardImageRefs returns every Markdown image src value found in a card's
// raw text fields.
func cardImageRefs(card types.Card) []string {
	cc := card.Content()
	switch cc.Kind() {
	case types.CardTypeBasic:
		refs := extractImageRefs(cc.Question)
		refs = append(refs, extractImageRefs(cc.Answer)...)
		return refs
	default: // CardTypeCloze
		return extractImageRefs(cc.Text)
	}
}

// extractImageRefs returns all "![](...)" src values from a Markdown string.
func extractImageRefs(src string) []string {
	var refs []string
	remaining := src
	for {
		start := strings.Index(remaining, "![")
		if start < 0 {
			break
		}
		rest := remaining[start+2:]
		closeBracket := strings.Index(rest, "]")
		if closeBracket < 0 {
			break
		}
		rest = rest[closeBracket+1:]
		if len(rest) == 0 || rest[0] != '(' {
			remaining = remaining[start+2:]
			continue
		}
		closeParen := strings.Index(rest[1:], ")")
		if closeParen < 0 {
			break
		}
		ref := rest[1 : closeParen+1]
		if ref != "" {
			refs = append(refs, ref)
		}
		remaining = rest[closeParen+2:]
	}
	return refs
}
