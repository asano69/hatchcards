package db

import (
	"github.com/asano69/hashcards/internal/cryptoutil"
	"github.com/asano69/hashcards/internal/errs"
	"github.com/pocketbase/pocketbase/core"
)

// ConnectionInput is the plaintext data accepted from the create/update API.
// Token is empty on update when the caller wants to keep the existing token.
type ConnectionInput struct {
	Name      string
	RemoteURL string
	Username  string
	Token     string
	LocalPath string
	Enabled   bool
}

// CreateConnection encrypts the token and inserts a new "connections" record.
func (db *Database) CreateConnection(in ConnectionInput) (*core.Record, error) {
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
	record.Set("local_path", in.LocalPath)
	record.Set("enabled", in.Enabled)
	if err := db.app.Save(record); err != nil {
		return nil, errs.Newf("save connection: %v", err)
	}
	return record, nil
}

// UpdateConnection updates a "connections" record by id. The token is only
// re-encrypted when in.Token is non-empty, so editing other fields doesn't
// require re-entering the secret.
func (db *Database) UpdateConnection(id string, in ConnectionInput) (*core.Record, error) {
	record, err := db.app.FindRecordById("connections", id)
	if err != nil {
		return nil, errs.Newf("find connection: %v", err)
	}

	record.Set("name", in.Name)
	record.Set("remote_url", in.RemoteURL)
	record.Set("username", in.Username)
	record.Set("local_path", in.LocalPath)
	record.Set("enabled", in.Enabled)

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
