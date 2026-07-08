// connections_api.go registers create/update for the "connections"
// collection. Listing, viewing, and deleting use PocketBase's generic
// Record API directly (safe because token_ciphertext is a hidden field);
// only create/update need to go through here so the plaintext token is
// encrypted before it ever reaches the database.
package serve

import (
	"github.com/asano69/hatchards/internal/db"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/sirupsen/logrus"

	"net/http"
)

// connectionRequest deliberately has no LocalPath field: it is always
// derived server-side from Name (db.SanitizeConnectionName), so a client
// can't set or override it, and never needs to.
type connectionRequest struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remote_url"`
	Username  string `json:"username"`
	Token     string `json:"token"`
	Enabled   bool   `json:"enabled"`
	// HookName is the name of a pre-installed post-sync hook script, or ""
	// for none. It is validated against hooksDir in db.CreateConnection /
	// db.UpdateConnection before being persisted.
	HookName string `json:"hook_name"`
}

func toConnectionInput(b connectionRequest) db.ConnectionInput {
	return db.ConnectionInput{
		Name: b.Name, RemoteURL: b.RemoteURL, Username: b.Username,
		Token: b.Token, Enabled: b.Enabled, HookName: b.HookName,
	}
}

// RegisterConnectionsAPI wires up the two encryption-sensitive endpoints.
// dataRoot is used to locate a connection's local data (dataRoot/<name> and
// dataRoot/.mirror/<name>) for cleanup when a connection is disabled.
// hooksDir is forwarded to db.CreateConnection / db.UpdateConnection so a
// connection's hook_name is validated against the operator-configured hooks
// directory before it is saved.
func RegisterConnectionsAPI(r *router.Router[*core.RequestEvent], database *db.Database, dataRoot, hooksDir string) {
	r.POST("/api/connections", func(e *core.RequestEvent) error {
		var body connectionRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("invalid request body", err)
		}
		record, err := database.CreateConnection(hooksDir, toConnectionInput(body))
		if err != nil {
			logrus.WithError(err).Warn("create connection failed")
			return e.BadRequestError("create connection failed", err)
		}
		return e.JSON(http.StatusOK, record)
	}).Bind(apis.RequireSuperuserAuth())

	r.PATCH("/api/connections/{id}", func(e *core.RequestEvent) error {
		var body connectionRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("invalid request body", err)
		}
		record, err := database.UpdateConnection(hooksDir, e.Request.PathValue("id"), toConnectionInput(body))
		if err != nil {
			logrus.WithError(err).Warn("update connection failed")
			return e.BadRequestError("update connection failed", err)
		}

		// Disabling a connection removes its local data even though the
		// record itself still exists (re-enabling triggers a fresh sync).
		if !body.Enabled {
			if err := removeConnectionData(dataRoot, body.Name); err != nil {
				logrus.WithError(err).Warn("cleanup connection data failed")
				return e.BadRequestError("cleanup connection data failed", err)
			}
		}

		return e.JSON(http.StatusOK, record)
	}).Bind(apis.RequireSuperuserAuth())

	// registerConnectionDeleteCleanup ensures local mirror data is always
	// removed when a "connections" record is deleted — regardless of
	// whether the deletion came through a dedicated route or PocketBase's
	// generic Record API (e.g. the frontend's
	// pb.collection("connections").delete(id), which never touches this
	// file's HTTP handlers). This mirrors the OnRecordCreate hook pattern
	// already used in internal/db/db.go for reviews -> cards syncing.
	registerConnectionDeleteCleanup(database, dataRoot)
}

// registerConnectionDeleteCleanup binds an OnRecordDelete hook for the
// "connections" collection. It reads the record's name before deletion (the
// mirror directory name is derived from Name, see db.SanitizeConnectionName)
// and removes the corresponding local directories after the delete commits.
// A cleanup failure is logged but does not roll back the delete — the
// connection record itself is still gone, matching the disable-cleanup
// behaviour above where a cleanup error is the exception, not the rule.
func registerConnectionDeleteCleanup(database *db.Database, dataRoot string) {
	database.App().OnRecordDelete("connections").BindFunc(func(e *core.RecordEvent) error {
		name := e.Record.GetString("name")

		if err := e.Next(); err != nil {
			return err
		}

		if err := removeConnectionData(dataRoot, name); err != nil {
			logrus.WithError(err).Warn("cleanup connection data failed after delete")
		}
		return nil
	})
}
