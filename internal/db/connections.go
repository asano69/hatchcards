package db

import (
	"regexp"
	"strings"

	"github.com/asano69/hashcards/internal/cryptoutil"
	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/hook"
	"github.com/pocketbase/pocketbase/core"

	"github.com/asano69/hashcards/internal/types"
)

// unsafePathCharsRe matches characters that are unsafe to use verbatim in a
// directory name (path separators, and other characters that are invalid or
// awkward on common filesystems).
var unsafePathCharsRe = regexp.MustCompile(`[\\/:*?"<>|]+`)

// SanitizeConnectionName converts a connection name into a filesystem-safe
// directory name. This is used as the connection's local_path so it never
// needs to be entered manually. Since connection names are unique (unique
// index on "name"), the derived directory names are unique too — this is
// what prevents two connections from ever being mirrored into the same
// directory (see docs/connections-mirror-design.md).
func SanitizeConnectionName(name string) string {
	cleaned := unsafePathCharsRe.ReplaceAllString(name, "_")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, ".") // avoid ".", "..", or leading-dot dirs
	if cleaned == "" {
		cleaned = "connection"
	}
	return cleaned
}

// ConnectionInput is the plaintext data accepted from the create/update API.
// Token is empty on update when the caller wants to keep the existing token.
// There is no LocalPath field: it is always derived server-side from Name
// (see SanitizeConnectionName), not supplied by the caller.
type ConnectionInput struct {
	Name      string
	RemoteURL string
	Username  string
	Token     string
	Enabled   bool
	// HookName is the name of a pre-installed post-sync script (see
	// internal/hook), or "" if this connection has no hook. It is
	// validated against hooksDir before being persisted, so a connection
	// can never reference a hook that doesn't exist or isn't executable.
	HookName string
}

// CreateConnection encrypts the token and inserts a new "connections" record.
// local_path is derived from the connection name at creation time. hooksDir
// is the operator-configured directory of runnable hook scripts; in.HookName
// is validated against it before the record is saved.
func (db *Database) CreateConnection(hooksDir string, in ConnectionInput) (*core.Record, error) {
	if _, err := hook.Resolve(hooksDir, in.HookName); err != nil {
		return nil, errs.Newf("invalid hook: %v", err)
	}

	collection, err := db.app.FindCollectionByNameOrId("connections")
	if err != nil {
		return nil, errs.Newf("find connections collection: %v", err)
	}
	ciphertext, err := cryptoutil.Encrypt([]byte(in.Token))
	if err != nil {
		return nil, errs.Newf("encrypt token: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("name", in.Name)
	record.Set("remote_url", in.RemoteURL)
	record.Set("username", in.Username)
	record.Set("token_ciphertext", ciphertext)
	record.Set("local_path", SanitizeConnectionName(in.Name))
	record.Set("enabled", in.Enabled)
	record.Set("hook_name", in.HookName)
	if err := db.app.Save(record); err != nil {
		return nil, errs.Newf("save connection: %v", err)
	}
	return record, nil
}

// UpdateConnection updates a "connections" record by id. The token is only
// re-encrypted when in.Token is non-empty, so editing other fields doesn't
// require re-entering the secret. hooksDir is used the same way as in
// CreateConnection, to validate in.HookName before saving.
//
// local_path is intentionally left untouched here, even if Name changes:
// it was fixed at creation time. Recomputing it on every rename would
// silently orphan the existing local clone (the mirror data would still be
// on disk under the old directory name, but no longer be found).
func (db *Database) UpdateConnection(hooksDir, id string, in ConnectionInput) (*core.Record, error) {
	if _, err := hook.Resolve(hooksDir, in.HookName); err != nil {
		return nil, errs.Newf("invalid hook: %v", err)
	}

	record, err := db.app.FindRecordById("connections", id)
	if err != nil {
		return nil, errs.Newf("find connection: %v", err)
	}

	record.Set("name", in.Name)
	record.Set("remote_url", in.RemoteURL)
	record.Set("username", in.Username)
	record.Set("enabled", in.Enabled)
	record.Set("hook_name", in.HookName)

	if in.Token != "" {
		ciphertext, err := cryptoutil.Encrypt([]byte(in.Token))
		if err != nil {
			return nil, errs.Newf("encrypt token: %v", err)
		}
		record.Set("token_ciphertext", ciphertext)
	}

	if err := db.app.Save(record); err != nil {
		return nil, errs.Newf("save connection: %v", err)
	}
	return record, nil
}

// EnsureLocalPath returns the connection's local_path, backfilling it from
// the connection's name if it is empty or ".". This self-heals connections
// created before local_path became a derived, non-editable field (including
// the ones that used to collide on the same directory). The backfilled
// value is persisted, so this only runs once per connection.
func (db *Database) EnsureLocalPath(id string) (string, error) {
	record, err := db.app.FindRecordById("connections", id)
	if err != nil {
		return "", errs.Newf("find connection: %v", err)
	}
	path := record.GetString("local_path")
	if path != "" && path != "." {
		return path, nil
	}
	path = SanitizeConnectionName(record.GetString("name"))
	record.Set("local_path", path)
	if err := db.app.Save(record); err != nil {
		return "", errs.Newf("backfill local_path: %v", err)
	}
	return path, nil
}

// DecryptConnectionToken decrypts a connection's token for one-off use (e.g.
// building a git remote URL). The caller must zero the result with
// cryptoutil.Zero once done.
func (db *Database) DecryptConnectionToken(id string) ([]byte, error) {
	record, err := db.app.FindRecordById("connections", id)
	if err != nil {
		return nil, errs.Newf("find connection: %v", err)
	}
	return cryptoutil.Decrypt(record.GetString("token_ciphertext"))
}

// MirrorableConnection holds a connection's plain (non-secret) fields, as
// needed by the mirror package. The token is fetched separately via
// DecryptConnectionToken so it's never bundled into a struct that outlives
// a single Sync call.
type MirrorableConnection struct {
	ID        string
	Name      string
	RemoteURL string
	Username  string
	LocalPath string
	// HookName is the post-sync hook to run after a successful mirror
	// sync, or "" if none is configured (see internal/hook).
	HookName string
}

// GetMirrorConnection returns the plain fields needed to mirror the
// connection with the given id.
func (db *Database) GetMirrorConnection(id string) (MirrorableConnection, error) {
	record, err := db.app.FindRecordById("connections", id)
	if err != nil {
		return MirrorableConnection{}, errs.Newf("find connection: %v", err)
	}
	return MirrorableConnection{
		ID:        record.Id,
		Name:      record.GetString("name"),
		RemoteURL: record.GetString("remote_url"),
		Username:  record.GetString("username"),
		LocalPath: record.GetString("local_path"),
		HookName:  record.GetString("hook_name"),
	}, nil
}

// ListEnabledConnections returns every connection with enabled = true, for
// use by a future "sync all" trigger.
func (db *Database) ListEnabledConnections() ([]MirrorableConnection, error) {
	records, err := db.app.FindRecordsByFilter("connections", "enabled = true", "", 0, 0, nil)
	if err != nil {
		return nil, errs.Newf("list enabled connections: %v", err)
	}
	out := make([]MirrorableConnection, 0, len(records))
	for _, r := range records {
		out = append(out, MirrorableConnection{
			ID:        r.Id,
			Name:      r.GetString("name"),
			RemoteURL: r.GetString("remote_url"),
			Username:  r.GetString("username"),
			LocalPath: r.GetString("local_path"),
			HookName:  r.GetString("hook_name"),
		})
	}
	return out, nil
}

// RecordSyncResult updates a connection's last_synced_at and last_error
// fields after a mirror attempt. On success (syncErr == nil), last_error is
// cleared and last_synced_at is set to now. On failure, last_synced_at is
// left untouched and last_error records the failure message, so it's
// visible on the Connections page without checking server logs.
func (db *Database) RecordSyncResult(id string, syncErr error) error {
	record, err := db.app.FindRecordById("connections", id)
	if err != nil {
		return errs.Newf("find connection: %v", err)
	}
	if syncErr != nil {
		record.Set("last_error", syncErr.Error())
	} else {
		record.Set("last_synced_at", types.Now().String())
		record.Set("last_error", "")
	}
	if err := db.app.Save(record); err != nil {
		return errs.Newf("record sync result: %v", err)
	}
	return nil
}
