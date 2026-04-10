// Package cache pre-renders the front and back HTML for every card in a drill
// session so the HTTP handlers can serve responses without re-running the
// Markdown renderer on every request.
//
// The rendered HTML includes the wrapper divs (.question, .answer, .prompt)
// and the .rich-text class, matching the structure that the Rust implementation
// builds in get.rs render_card().
package cache

import (
	"fmt"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/markdown"
	"github.com/asano69/hashcards/internal/types"
)

// Entry holds the pre-rendered HTML for one card.
//
// Front is the HTML shown before the answer is revealed.
// Back is the HTML shown after the answer is revealed.
//
// Both strings include the wrapper divs (.question/.answer/.prompt) and the
// .rich-text class so that CSS rules apply without any further wrapping in the
// template.
type Entry struct {
	Front string
	Back  string
}

// Cache maps a CardHash to its pre-rendered HTML entry.
type Cache struct {
	entries map[types.CardHash]Entry
}

// Build renders the front and back HTML for every due card and stores the
// results in a new Cache. Cards whose rendering fails are silently skipped;
// the handler will fall back gracefully if the hash is absent.
func Build(due []collection.DueCard) *Cache {
	c := &Cache{entries: make(map[types.CardHash]Entry, len(due))}
	for _, dc := range due {
		card := dc.Card
		deckFilePath := card.FilePath()

		frontContent, err := markdown.HTMLFront(card, deckFilePath)
		if err != nil {
			continue
		}
		backContent, err := markdown.HTMLBack(card, deckFilePath)
		if err != nil {
			continue
		}

		var front, back string
		switch card.CardType() {
		case types.CardTypeBasic:
			// Front (pre-reveal): question visible, answer area empty.
			front = fmt.Sprintf(
				`<div class="question rich-text">%s</div><div class="answer rich-text"></div>`,
				frontContent,
			)
			// Back (post-reveal): question and answer both visible.
			back = fmt.Sprintf(
				`<div class="question rich-text">%s</div><div class="answer rich-text">%s</div>`,
				frontContent, backContent,
			)
		default: // CardTypeCloze
			front = fmt.Sprintf(`<div class="prompt rich-text">%s</div>`, frontContent)
			back = fmt.Sprintf(`<div class="prompt rich-text">%s</div>`, backContent)
		}

		c.entries[card.Hash()] = Entry{Front: front, Back: back}
	}
	return c
}

// Get returns the cached Entry for the given hash, and whether it was found.
func (c *Cache) Get(hash types.CardHash) (Entry, bool) {
	e, ok := c.entries[hash]
	return e, ok
}

// Len returns the number of entries in the cache.
func (c *Cache) Len() int {
	return len(c.entries)
}
