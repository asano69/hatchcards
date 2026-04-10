// Package stats implements the "stats" subcommand, which prints a summary
// of card counts and review activity for the collection.
package stats

import (
	"fmt"
	"io"

	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/types"
)

// Run opens the database at dbPath, loads the collection at root, and writes
// a human-readable stats summary to out.
func Run(root, dbPath string, out io.Writer) error {
	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	col, err := collection.Load(root, database)
	if err != nil {
		return err
	}

	today := types.Today()
	stats, err := col.Stats(today)
	if err != nil {
		return err
	}

	reviewsToday, err := database.CountReviewsInDate(today)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Total cards:    %d\n", stats.TotalCards)
	fmt.Fprintf(out, "New cards:      %d\n", stats.NewCards)
	fmt.Fprintf(out, "Reviewed cards: %d\n", stats.ReviewedCards)
	fmt.Fprintf(out, "Due today:      %d\n", stats.DueToday)
	fmt.Fprintf(out, "Reviews today:  %d\n", reviewsToday)

	return nil
}
