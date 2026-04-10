// Package check implements the "check" subcommand, which parses every deck
// file in the collection and validates that all referenced media files exist
// and are within the accepted size limit.
package check

import (
	"fmt"
	"io"
	"strings"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/markdown"
	"github.com/asano69/hashcards/internal/media"
	"github.com/asano69/hashcards/internal/types"
)

// Result is the outcome of a single check run.
type Result struct {
	// CardCount is the total number of cards successfully parsed.
	CardCount int
	// Errors holds one entry per validation problem found.
	Errors []string
}

// OK returns true when no validation errors were found.
func (r Result) OK() bool { return len(r.Errors) == 0 }

// Run loads the collection at root (using an in-memory database so no
// persistent state is written) and validates every card's rendered HTML and
// every media reference it contains.
func Run(root string, out io.Writer) (Result, error) {
	database, err := db.Open(":memory:")
	if err != nil {
		return Result{}, err
	}
	defer database.Close()

	col, err := collection.Load(root, database)
	if err != nil {
		return Result{}, err
	}

	var errors []string

	for _, card := range col.Cards {
		deckFilePath := card.FilePath()

		// Render both faces; errors here indicate broken Markdown.
		if _, err := markdown.HTMLFront(card, deckFilePath); err != nil {
			errors = append(errors, fmt.Sprintf(
				"%s:%d: error rendering front: %v",
				deckFilePath, card.LineStart(), err,
			))
		}
		if _, err := markdown.HTMLBack(card, deckFilePath); err != nil {
			errors = append(errors, fmt.Sprintf(
				"%s:%d: error rendering back: %v",
				deckFilePath, card.LineStart(), err,
			))
		}

		// Validate every image reference found in the card's raw Markdown text.
		for _, ref := range cardImageRefs(card) {
			resolved, err := media.Resolve(deckFilePath, ref)
			if err != nil {
				errors = append(errors, fmt.Sprintf(
					"%s:%d: %v", deckFilePath, card.LineStart(), err,
				))
				continue
			}
			if ve := media.Validate(resolved); ve != nil {
				errors = append(errors, fmt.Sprintf(
					"%s:%d: %v", deckFilePath, card.LineStart(), ve,
				))
			}
		}
	}

	for _, e := range errors {
		fmt.Fprintln(out, e)
	}

	return Result{CardCount: len(col.Cards), Errors: errors}, nil
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
