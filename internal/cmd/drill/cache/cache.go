// Package cache pre-renders the front and back HTML for every card in a drill
// session so the HTTP handlers can serve responses without re-running the
// Markdown renderer on every request.
package cache

import (
	"fmt"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/markdown"
	"github.com/asano69/hashcards/internal/types"
)

// Entry holds the pre-rendered HTML for one card.
type Entry struct {
	Front string
	Back  string
}

// Cache maps a CardHash to its pre-rendered HTML entry.
type Cache struct {
	entries map[types.CardHash]Entry
}

// Build renders the front and back HTML for every due card and stores the
// results in a new Cache.
// fileMountBase is the URL prefix for the /file/ handler (e.g. "/drill/geo").
func Build(due []collection.DueCard, fileMountBase string) *Cache {
	c := &Cache{entries: make(map[types.CardHash]Entry, len(due))}
	for _, dc := range due {
		card := dc.Card
		deckFilePath := card.FilePath()

		frontContent, err := markdown.HTMLFront(card, deckFilePath, fileMountBase)
		if err != nil {
			continue
		}
		backContent, err := markdown.HTMLBack(card, deckFilePath, fileMountBase)
		if err != nil {
			continue
		}

		var front, back string
		switch card.CardType() {
		case types.CardTypeBasic:
			front = fmt.Sprintf(
				`<div class="question rich-text">%s</div><div class="answer rich-text"></div>`,
				frontContent,
			)
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
