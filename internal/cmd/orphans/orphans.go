// Package orphans implements the "orphans" subcommand, which lists or deletes
// cards that exist in the performance database but are no longer present in
// any deck file — matching the Rust implementation in cmd/orphans.rs.
package orphans

import (
	"fmt"
	"io"
	"sort"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/types"
)

// List prints the hex hash of every orphan card, one per line.
// An orphan card is one that exists in the database but not in the collection.
func List(root, dbPath string, out io.Writer) error {
	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	orphans, err := getOrphans(root, database)
	if err != nil {
		return err
	}

	for _, h := range orphans {
		fmt.Fprintln(out, h.Hex())
	}
	return nil
}

// Delete removes every orphan card from the database and prints each deleted
// hash, one per line.
func Delete(root, dbPath string, out io.Writer) error {
	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	orphans, err := getOrphans(root, database)
	if err != nil {
		return err
	}

	for _, h := range orphans {
		if err := database.DeleteCard(h); err != nil {
			return err
		}
		fmt.Fprintln(out, h.Hex())
	}
	return nil
}

// getOrphans returns card hashes that are in the database but not in the
// collection, sorted for deterministic output.
func getOrphans(root string, database *db.Database) ([]types.CardHash, error) {
	// Use an in-memory DB just to load the collection without side effects on
	// the real database; pass the real database for hash lookups below.
	memDB, err := db.Open(":memory:")
	if err != nil {
		return nil, err
	}
	defer memDB.Close()

	col, err := collection.Load(root, memDB)
	if err != nil {
		return nil, err
	}

	// Build the set of hashes currently on disk.
	collHashes := make(map[types.CardHash]struct{}, len(col.Cards))
	for _, card := range col.Cards {
		collHashes[card.Hash()] = struct{}{}
	}

	// Cards in the real database that are absent from the collection are orphans.
	dbHashes, err := database.CardHashes()
	if err != nil {
		return nil, err
	}

	var orphans []types.CardHash
	for h := range dbHashes {
		if _, exists := collHashes[h]; !exists {
			orphans = append(orphans, h)
		}
	}

	// Sort for consistent, deterministic output.
	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].Less(orphans[j])
	})
	return orphans, nil
}
