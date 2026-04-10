// Package collection walks a directory tree of deck files, synchronises the
// cards it finds with the performance database, and answers queries about
// which cards are due for review today.
package collection

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/parser"
	"github.com/asano69/hashcards/internal/types"
)

// DueCard pairs a card with its current performance record.
type DueCard struct {
	Card        types.Card
	Performance types.Performance
}

// Stats holds aggregate counts about the collection.
type Stats struct {
	TotalCards    int
	NewCards      int
	ReviewedCards int
	DueToday      int
}

// Collection represents a loaded set of decks and their performance data.
type Collection struct {
	// Root is the absolute path of the collection directory.
	Root string
	// Cards is every card found across all deck files, in stable order.
	Cards []types.Card
	// DB is the open performance database.
	DB *db.Database
}

// Load walks root for "*.md" files, parses every deck it finds, inserts any
// new cards into database, and returns the populated Collection.
// Cards that exist in the database but are no longer present in any deck file
// are removed from the database.
func Load(root string, database *db.Database) (*Collection, error) {
	cards, err := walkDecks(root)
	if err != nil {
		return nil, err
	}

	if err := syncDB(cards, database); err != nil {
		return nil, err
	}

	return &Collection{Root: root, Cards: cards, DB: database}, nil
}

// DueToday returns every card that is due for review on today's date, paired
// with its current performance record. Cards are returned in a stable order
// (sorted by hash hex string) so callers can shuffle them deterministically.
func (c *Collection) DueToday(today types.Date) ([]DueCard, error) {
	dueHashes, err := c.DB.DueToday(today)
	if err != nil {
		return nil, err
	}

	var due []DueCard
	for _, card := range c.Cards {
		if _, ok := dueHashes[card.Hash()]; !ok {
			continue
		}
		perf, err := c.DB.GetCardPerformance(card.Hash())
		if err != nil {
			return nil, err
		}
		due = append(due, DueCard{Card: card, Performance: perf})
	}

	// Sort by hash for a stable, deterministic order before any shuffle.
	sort.Slice(due, func(i, j int) bool {
		return due[i].Card.Hash().Less(due[j].Card.Hash())
	})
	return due, nil
}

// Stats returns aggregate counts for the collection as of today.
func (c *Collection) Stats(today types.Date) (Stats, error) {
	dueHashes, err := c.DB.DueToday(today)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{TotalCards: len(c.Cards)}
	for _, card := range c.Cards {
		perf, err := c.DB.GetCardPerformance(card.Hash())
		if err != nil {
			return Stats{}, err
		}
		if perf.IsNew() {
			stats.NewCards++
		} else {
			stats.ReviewedCards++
		}
		if _, ok := dueHashes[card.Hash()]; ok {
			stats.DueToday++
		}
	}
	return stats, nil
}

// CardByHash returns the card with the given hash, or an error if not found.
func (c *Collection) CardByHash(hash types.CardHash) (types.Card, error) {
	for _, card := range c.Cards {
		if card.Hash().Equal(hash) {
			return card, nil
		}
	}
	return types.Card{}, errs.Newf("card not found: %s", hash)
}

// walkDecks recursively finds all "*.md" files under root, parses each one,
// and returns all cards in a stable order (sorted by file path then card hash).
func walkDecks(root string) ([]types.Card, error) {
	var deckPaths []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ".md" {
			deckPaths = append(deckPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, errs.Newf("walk collection root %s: %v", root, err)
	}

	// Sort paths for stable cross-platform ordering.
	sort.Strings(deckPaths)

	var allCards []types.Card
	for _, path := range deckPaths {
		cards, err := parser.ParseFile(path)
		if err != nil {
			return nil, err
		}
		allCards = append(allCards, cards...)
	}
	return allCards, nil
}

// syncDB ensures the database reflects the current set of cards on disk.
// New cards are inserted; cards no longer on disk are deleted.
func syncDB(cards []types.Card, database *db.Database) error {
	// Build a set of hashes currently on disk.
	onDisk := make(map[types.CardHash]struct{}, len(cards))
	for _, card := range cards {
		onDisk[card.Hash()] = struct{}{}
	}

	// Fetch hashes currently in the database.
	inDB, err := database.CardHashes()
	if err != nil {
		return err
	}

	now := types.Now()

	// Insert cards that are on disk but not in the database.
	for _, card := range cards {
		if _, exists := inDB[card.Hash()]; !exists {
			if err := database.InsertCard(card.Hash(), now); err != nil {
				return err
			}
		}
	}

	// Delete cards that are in the database but no longer on disk.
	for hash := range inDB {
		if _, exists := onDisk[hash]; !exists {
			if err := database.DeleteCard(hash); err != nil {
				return err
			}
		}
	}

	return nil
}
