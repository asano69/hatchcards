// mirror_api.go registers the manual mirror-trigger endpoint. Actually
// running go-git lives in internal/mirror; this file only wires the HTTP
// route to it and handles the decrypt/zero lifecycle of the token.
package serve

import (
	"net/http"
	"path/filepath"

	"github.com/asano69/hashcards/internal/cryptoutil"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/mirror"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/sirupsen/logrus"
)

// RegisterMirrorAPI registers the endpoint that manually triggers a mirror
// sync for one connection. dataRoot is used to resolve a connection's
// local_path when it isn't already an absolute path.
func RegisterMirrorAPI(r *router.Router[*core.RequestEvent], database *db.Database, dataRoot string) {
	r.POST("/api/connections/{id}/mirror", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if err := syncConnection(database, dataRoot, id); err != nil {
			return e.BadRequestError("mirror sync failed", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"synced": true})
	}).Bind(apis.RequireSuperuserAuth())
}

// syncConnection decrypts the connection's token just long enough to run
// one Sync call, zeroing it immediately afterwards regardless of outcome,
// then persists the result (success or error) back onto the record.
func syncConnection(database *db.Database, dataRoot, id string) error {
	mc, err := database.GetMirrorConnection(id)
	if err != nil {
		return err
	}
	if mc.LocalPath == "" {
		return errs.Newf("connection %q has no local_path configured", mc.Name)
	}

	token, err := database.DecryptConnectionToken(id)
	if err != nil {
		return err
	}
	defer cryptoutil.Zero(token)

	localPath := mc.LocalPath
	if !filepath.IsAbs(localPath) {
		localPath = filepath.Join(dataRoot, localPath)
	}

	syncErr := mirror.Sync(mirror.Connection{
		Name:      mc.Name,
		RemoteURL: mc.RemoteURL,
		Username:  mc.Username,
		LocalPath: localPath,
	}, token)

	// Always record the outcome, even if syncErr is non-nil, so last_error
	// is visible in the UI. A failure to persist the result is logged but
	// doesn't mask the original syncErr, which is what the caller sees.
	if recordErr := database.RecordSyncResult(id, syncErr); recordErr != nil {
		logrus.WithError(recordErr).Warn("record sync result failed")
	}
	return syncErr
}
