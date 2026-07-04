package migrations

import (
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	core.AppMigrations.Register(func(app core.App) error {
		collection := core.NewBaseCollection("reviews")
		collection.Fields.Add(
			&core.TextField{Name: "session_id", Required: true},
			&core.TextField{Name: "card_hash", Required: true},
			&core.TextField{Name: "reviewed_at", Required: true},
			&core.TextField{Name: "grade", Required: true},
			&core.NumberField{Name: "stability", Required: true},
			&core.NumberField{Name: "difficulty", Required: true},
			&core.NumberField{Name: "interval_raw", Required: true},
			&core.NumberField{Name: "interval_days", OnlyInt: true, Required: true},
			&core.TextField{Name: "due_date", Required: true},
		)
		collection.AddIndex("idx_reviews_session_id", false, "session_id", "")
		collection.AddIndex("idx_reviews_card_hash", false, "card_hash", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
