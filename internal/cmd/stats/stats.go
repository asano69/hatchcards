package stats

import (
	"fmt"
	"io"

	"github.com/pocketbase/pocketbase"

	"github.com/asano69/hatchards/internal/collection"
	"github.com/asano69/hatchards/internal/db"
	"github.com/asano69/hatchards/internal/types"
)

func Run(app *pocketbase.PocketBase, root string, out io.Writer) error {
	database, err := db.New(app)
	if err != nil {
		return err
	}

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
