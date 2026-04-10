// Package cmdutil provides small helpers shared by the command packages.
package cmdutil

import (
	"strings"

	"github.com/asano69/hashcards/internal/types"
)

// CardImageRefs returns every Markdown image src value ("![](...)")
// found in the raw text fields of a card.
func CardImageRefs(card types.Card) []string {
	cc := card.Content()
	switch cc.Kind() {
	case types.CardTypeBasic:
		refs := ExtractImageRefs(cc.Question)
		refs = append(refs, ExtractImageRefs(cc.Answer)...)
		return refs
	default: // CardTypeCloze
		return ExtractImageRefs(cc.Text)
	}
}

// ExtractImageRefs returns all "![](...)" src values from a Markdown string.
// This is a lightweight extraction used only for pre-render validation;
// the actual Markdown parse is handled by the markdown package.
func ExtractImageRefs(src string) []string {
	var refs []string
	remaining := src
	for {
		start := strings.Index(remaining, "![")
		if start < 0 {
			break
		}
		rest := remaining[start+2:]

		// Skip the alt text inside [...].
		closeBracket := strings.Index(rest, "]")
		if closeBracket < 0 {
			break
		}
		rest = rest[closeBracket+1:]

		// Expect "(" immediately after "]".
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
