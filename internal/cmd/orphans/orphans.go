// Package orphans implements the "orphans" subcommand, which lists or deletes
// cards that exist in the performance database but are no longer present in
// any deck file — matching the Rust implementation in cmd/orphans.rs
package orphans

import (
	"fmt"
	"io"
	"sort"

	"github.com/pocketbase/pocketbase"

	"github.com/asano69/hatchards/internal/collection"
	"github.com/asano69/hatchards/internal/db"
	"github.com/asano69/hatchards/internal/types"
)

func List(app *pocketbase.PocketBase, root string, out io.Writer) error {
	database, err := db.New(app)
	if err != nil {
		return err
	}

	orphans, err := getOrphans(root, database)
	if err != nil {
		return err
	}

	for _, h := range orphans {
		fmt.Fprintln(out, h.Hex())
	}
	return nil
}

func Delete(app *pocketbase.PocketBase, root string, out io.Writer) error {
	database, err := db.New(app)
	if err != nil {
		return err
	}

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

// getOrphans is unchanged except for one call:
func getOrphans(root string, database *db.Database) ([]types.CardHash, error) {
	memDB, err := db.OpenScratch()
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
