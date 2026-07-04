package migrations

import (
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	core.AppMigrations.Register(func(app core.App) error {
		collection := core.NewBaseCollection("sessions")
		collection.Fields.Add(
			&core.TextField{Name: "started_at", Required: true},
			&core.TextField{Name: "ended_at", Required: true},
		)
		collection.AddIndex("idx_sessions_started_at", false, "started_at", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("sessions")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
