// mirror_api.go registers the manual mirror-trigger endpoint. Actually
// running go-git lives in internal/mirror; running a post-sync hook lives
// in internal/hook. This file only wires the HTTP route to both and handles
// the decrypt/zero lifecycle of the token.
package serve

import (
	"context"
	"github.com/asano69/hashcards/internal/cryptoutil"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/hook"
	"github.com/asano69/hashcards/internal/mirror"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/sirupsen/logrus"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// hookTimeout bounds how long a single post-sync hook script may run, so a
// runaway script can't block the mirror sync endpoint indefinitely.
const hookTimeout = 5 * time.Minute

// RegisterMirrorAPI registers the endpoint that manually triggers a mirror
// sync for one connection. dataRoot is used to derive a connection's mirror
// directory (dataRoot/<name> or dataRoot/.mirror/<name>, see mirrorRoot),
// and (together with the connection name) to derive the output directory
// for any post-sync hook. hooksDir is the operator-configured directory of
// runnable hook scripts (see internal/hook).
func RegisterMirrorAPI(r *router.Router[*core.RequestEvent], database *db.Database, dataRoot, hooksDir string) {
	r.POST("/api/connections/{id}/mirror", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		logrus.WithField("connection_id", id).Info("mirror sync: request received")
		if err := syncConnection(database, dataRoot, hooksDir, id); err != nil {
			logrus.WithField("connection_id", id).WithError(err).Warn("mirror sync: request failed")
			return e.BadRequestError("mirror sync failed", err)
		}
		logrus.WithField("connection_id", id).Info("mirror sync: request succeeded")
		return e.JSON(http.StatusOK, map[string]any{"synced": true})
	}).Bind(apis.RequireSuperuserAuth())
}

// mirrorRoot returns the directory a connection's sanitized name is
// resolved against for cloning/pulling.
//
// Connections without a hook mirror directly into dataRoot (e.g.
// cards/<name>), matching the pre-hook behavior exactly.
//
// Connections with a hook mirror into dataRoot/.mirror instead: their git
// working tree holds source notes, not JSON decks, so it must stay out of
// the deck directory that walkDecks scans. The hook's generated JSON decks
// go directly into dataRoot/<name> (see runPostSyncHook), which is scanned
// as usual.
func mirrorRoot(dataRoot string, mc db.MirrorableConnection) string {
	if mc.HookName == "" {
		return dataRoot
	}
	return filepath.Join(dataRoot, ".mirror")
}

// syncConnection decrypts the connection's token just long enough to run
// one Sync call, zeroing it immediately afterwards regardless of outcome,
// then persists the result (success or error) back onto the record.
//
// The mirror directory is always computed from mc.Name via
// db.SanitizeConnectionName — it is never user-supplied and never read from
// storage. Which root it resolves under depends on whether a hook is
// configured; see mirrorRoot.
//
// If the connection has a hook_name, it runs once the sync succeeds,
// reading from the freshly-synced working tree (dataRoot/.mirror/<name>)
// and writing generated JSON decks to dataRoot/<name>, which walkDecks
// already picks up like any other deck directory. Connections without a
// hook_name are unaffected: the hook step is skipped entirely, exactly as
// before this feature existed.
func syncConnection(database *db.Database, dataRoot, hooksDir, id string) error {
	mc, err := database.GetMirrorConnection(id)
	if err != nil {
		return err
	}

	token, err := database.DecryptConnectionToken(id)
	if err != nil {
		return err
	}
	defer cryptoutil.Zero(token)

	localPath := filepath.Join(mirrorRoot(dataRoot, mc), db.SanitizeConnectionName(mc.Name))

	logrus.WithFields(logrus.Fields{
		"connection":       mc.Name,
		"remote_url":       mc.RemoteURL,
		"local_path_final": localPath,
		"data_root":        dataRoot,
	}).Info("mirror sync: resolved local path")

	syncErr := mirror.Sync(mirror.Connection{
		Name:      mc.Name,
		RemoteURL: mc.RemoteURL,
		Username:  mc.Username,
		LocalPath: localPath,
	}, token)

	if syncErr == nil && mc.HookName != "" {
		syncErr = runPostSyncHook(hooksDir, mc, dataRoot, localPath)
	}

	// Always record the outcome, even if syncErr is non-nil, so last_error
	// is visible in the UI. A failure to persist the result is logged but
	// doesn't mask the original syncErr, which is what the caller sees.
	if recordErr := database.RecordSyncResult(id, syncErr); recordErr != nil {
		logrus.WithError(recordErr).Warn("mirror sync: record sync result failed")
	}
	return syncErr
}

// runPostSyncHook resolves and runs the connection's configured hook
// script, writing its output to dataRoot/<sanitized name> — the
// connection's normal deck directory — rather than into the git working
// tree itself (which lives under dataRoot/.mirror/<sanitized name>). This
// also means a generated file can never collide with an untracked path on
// the next pull, since the two directories are entirely separate trees.
func runPostSyncHook(hooksDir string, mc db.MirrorableConnection, dataRoot, sourceDir string) error {
	scriptPath, err := hook.Resolve(hooksDir, mc.HookName)
	if err != nil {
		return err
	}
	outputDir := filepath.Join(dataRoot, db.SanitizeConnectionName(mc.Name))

	log := logrus.WithFields(logrus.Fields{
		"connection": mc.Name,
		"hook":       mc.HookName,
		"output_dir": outputDir,
	})
	log.Info("post-sync hook: running")

	ctx, cancel := context.WithTimeout(context.Background(), hookTimeout)
	defer cancel()
	out, err := hook.Run(ctx, scriptPath, sourceDir, outputDir)

	logOutput := func(entry *logrus.Entry, out string) {
		out = strings.TrimRight(out, "\n")
		if out == "" {
			return
		}
		for _, line := range strings.Split(out, "\n") {
			entry.Info(line)
		}
	}

	if err != nil {
		log.WithError(err).Warn("post-sync hook: failed")
		logOutput(log, out)
		return err
	}
	log.Info("post-sync hook: succeeded")
	logOutput(log, out)
	return nil
}
