// Package cache pre-renders the front and back HTML for every card in a drill
// session so the HTTP handlers can serve responses without re-running the
// Markdown renderer on every request.
package cache

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/asano69/hatchards/internal/collection"
	"github.com/asano69/hatchards/internal/markdown"
	"github.com/asano69/hatchards/internal/types"
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

// cacheJob is one unit of work: render a single due card into an Entry.
type cacheJob struct {
	card collection.DueCard
}

// cacheResult pairs a card hash with its rendered entry, produced by a
// worker goroutine and consumed by the single goroutine that owns the
// entries map (so no locking is needed around the map itself).
type cacheResult struct {
	hash  types.CardHash
	entry Entry
}

// Build renders the front and back HTML for every due card and stores the
// results in a new Cache. Rendering one card is a pure function of that
// card's content, independent of every other card, so the work is fanned
// out across a small worker pool instead of running strictly sequentially.
// This mainly matters for large due lists, where session start/reset used
// to block on rendering every card one at a time.
// collectionRoot is the absolute path of the collection root, used to
// resolve media references to the URL the drill server actually serves them
// from (see markdown.HTMLFront/HTMLBack). fileMountBase is the URL prefix
// for the /file/ handler (e.g. "/drill/geo").
func Build(due []collection.DueCard, collectionRoot, fileMountBase string) *Cache {
	c := &Cache{entries: make(map[types.CardHash]Entry, len(due))}
	if len(due) == 0 {
		return c
	}

	jobs := make(chan cacheJob)
	results := make(chan cacheResult)

	workers := runtime.NumCPU()
	if workers > len(due) {
		workers = len(due)
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				if entry, ok := renderEntry(j.card.Card, collectionRoot, fileMountBase); ok {
					results <- cacheResult{hash: j.card.Card.Hash(), entry: entry}
				}
			}
		}()
	}

	// Feed jobs from a separate goroutine so this function doesn't deadlock
	// once the (unbuffered) jobs channel fills up before workers start
	// draining it.
	go func() {
		for _, dc := range due {
			jobs <- cacheJob{card: dc}
		}
		close(jobs)
	}()

	// Close results once every worker is done, so the range below
	// terminates.
	go func() {
		wg.Wait()
		close(results)
	}()

	// The map is only ever written here, in the calling goroutine, so it
	// needs no mutex.
	for r := range results {
		c.entries[r.hash] = r.entry
	}
	return c
}

// renderEntry renders both faces of a single card into an Entry. It returns
// ok=false if either face fails to render, matching Build's previous
// behaviour of skipping cards with rendering errors.
func renderEntry(card types.Card, collectionRoot, fileMountBase string) (Entry, bool) {
	deckFilePath := card.FilePath()

	frontContent, err := markdown.HTMLFront(card, deckFilePath, collectionRoot, fileMountBase)
	if err != nil {
		return Entry{}, false
	}
	backContent, err := markdown.HTMLBack(card, deckFilePath, collectionRoot, fileMountBase)
	if err != nil {
		return Entry{}, false
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

	return Entry{Front: front, Back: back}, true
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
