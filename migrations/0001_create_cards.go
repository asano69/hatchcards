// Package migrations defines the PocketBase schema as Go migrations.
// PocketBase applies any migration that hasn't run yet automatically during
// app.Bootstrap(), tracking progress in its own "_migrations" collection.
// Importing this package for its side effects (see internal/db/db.go) is
// enough to keep the schema in sync — no manual schema code is needed.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	core.AppMigrations.Register(func(app core.App) error {
		collection := core.NewBaseCollection("cards")
		collection.Fields.Add(
			&core.TextField{Name: "card_hash", Required: true, Presentable: true},
			&core.TextField{Name: "added_at", Required: true},
			&core.TextField{Name: "last_reviewed_at"},
			&core.NumberField{Name: "stability"},
			&core.NumberField{Name: "difficulty"},
			&core.NumberField{Name: "interval_raw"},
			&core.NumberField{Name: "interval_days", OnlyInt: true},
			&core.TextField{Name: "due_date"},
			&core.NumberField{Name: "review_count", OnlyInt: true},
		)
		collection.AddIndex("idx_cards_card_hash_unique", true, "card_hash", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("cards")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
