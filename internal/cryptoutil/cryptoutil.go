// Package cryptoutil provides AES-256-GCM encryption for secrets that must
// be stored at rest (e.g. git remote tokens) and decrypted only in memory
// when actually needed.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"

	"github.com/asano69/hatchards/internal/errs"
)

// masterKeyEnv holds the base64-encoded 32-byte AES-256 master key.
const masterKeyEnv = "HATCHARDS_MASTER_KEY"

// loadMasterKey reads and decodes the master key on every call, rather than
// caching it, so it never lingers in a package-level variable longer than
// a single Encrypt/Decrypt.
func loadMasterKey() ([]byte, error) {
	encoded := os.Getenv(masterKeyEnv)
	if encoded == "" {
		return nil, errs.Newf("%s is not set", masterKeyEnv)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errs.Newf("%s is not valid base64: %v", masterKeyEnv, err)
	}
	if len(key) != 32 {
		return nil, errs.Newf("%s must decode to 32 bytes, got %d", masterKeyEnv, len(key))
	}
	return key, nil
}

// Encrypt seals plaintext with AES-256-GCM and returns base64(nonce || ciphertext).
func Encrypt(plaintext []byte) (string, error) {
	key, err := loadMasterKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errs.Newf("create cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errs.Newf("create GCM: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", errs.Newf("generate nonce: %v", err)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. The caller must zero the returned slice with
// Zero once the secret is no longer needed.
func Decrypt(ciphertextB64 string) ([]byte, error) {
	key, err := loadMasterKey()
	if err != nil {
		return nil, err
	}
	sealed, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, errs.Newf("decode ciphertext: %v", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errs.Newf("create cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errs.Newf("create GCM: %v", err)
	}
	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errs.Newf("decrypt: %v", err)
	}
	return plaintext, nil
}

// Zero overwrites b with zeros in place.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
