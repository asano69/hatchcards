// connections_api.go registers create/update for the "connections"
// collection. Listing, viewing, and deleting use PocketBase's generic
// Record API directly (safe because token_ciphertext is a hidden field);
// only create/update need to go through here so the plaintext token is
// encrypted before it ever reaches the database.
package serve

import (
	"net/http"

	"github.com/asano69/hashcards/internal/db"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

type connectionRequest struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remote_url"`
	Username  string `json:"username"`
	Token     string `json:"token"`
	LocalPath string `json:"local_path"`
	Enabled   bool   `json:"enabled"`
}

func toConnectionInput(b connectionRequest) db.ConnectionInput {
	return db.ConnectionInput{
		Name: b.Name, RemoteURL: b.RemoteURL, Username: b.Username,
		Token: b.Token, LocalPath: b.LocalPath, Enabled: b.Enabled,
	}
}

// RegisterConnectionsAPI wires up the two encryption-sensitive endpoints.
func RegisterConnectionsAPI(r *router.Router[*core.RequestEvent], database *db.Database) {
	r.POST("/api/connections", func(e *core.RequestEvent) error {
		var body connectionRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("invalid request body", err)
		}
		record, err := database.CreateConnection(toConnectionInput(body))
		if err != nil {
			return e.BadRequestError("create connection failed", err)
		}
		return e.JSON(http.StatusOK, record)
	}).Bind(apis.RequireSuperuserAuth())

	r.PATCH("/api/connections/{id}", func(e *core.RequestEvent) error {
		var body connectionRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("invalid request body", err)
		}
		record, err := database.UpdateConnection(e.Request.PathValue("id"), toConnectionInput(body))
		if err != nil {
			return e.BadRequestError("update connection failed", err)
		}
		return e.JSON(http.StatusOK, record)
	}).Bind(apis.RequireSuperuserAuth())
}
