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
// directory name. The mirror destination for a connection is always derived
// from its name at the moment it's needed (dataRoot/<name> or
// dataRoot/.mirror/<name>) rather than stored, so two connections can never
// collide on the same directory as long as their names differ (names are
// unique via a unique index).
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
// hooksDir is the operator-configured directory of runnable hook scripts;
// in.HookName is validated against it before the record is saved.
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
//
// There is no LocalPath field: the mirror destination is always derived
// from Name (via SanitizeConnectionName) at the point of use, not stored.
type MirrorableConnection struct {
	ID        string
	Name      string
	RemoteURL string
	Username  string
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
