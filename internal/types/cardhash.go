package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/asano69/hashcards/internal/errs"
	"github.com/zeebo/blake3"
)

const hashSize = 32

// CardHash is a blake3 digest that uniquely identifies a card's content.
// It is comparable and can be used as a map key.
type CardHash struct {
	b [hashSize]byte
}

// HashBytes computes the blake3 hash of the given byte slice.
func HashBytes(data []byte) CardHash {
	digest := blake3.Sum256(data)
	return CardHash{b: digest}
}

// ParseCardHash parses a 64-character lowercase hex string into a CardHash.
func ParseCardHash(s string) (CardHash, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != hashSize {
		return CardHash{}, errs.New("invalid hash in performance database")
	}
	var h CardHash
	copy(h.b[:], b)
	return h, nil
}

// Hex returns the lowercase hex-encoded hash string.
func (h CardHash) Hex() string {
	return hex.EncodeToString(h.b[:])
}

// String implements fmt.Stringer, returning the hex-encoded hash.
func (h CardHash) String() string {
	return h.Hex()
}

// Equal returns true if h and other are the same hash.
func (h CardHash) Equal(other CardHash) bool {
	return h.b == other.b
}

// Less returns true if h comes before other in lexicographic byte order.
func (h CardHash) Less(other CardHash) bool {
	return bytes.Compare(h.b[:], other.b[:]) < 0
}

// MarshalJSON serializes CardHash as a hex JSON string.
func (h CardHash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Hex())
}

// UnmarshalJSON deserializes a CardHash from a hex JSON string.
func (h *CardHash) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseCardHash(s)
	if err != nil {
		return err
	}
	*h = parsed
	return nil
}

// Hasher computes a CardHash incrementally from multiple pieces of data.
type Hasher struct {
	inner *blake3.Hasher
}

// NewHasher creates a new Hasher ready to accept data.
func NewHasher() *Hasher {
	return &Hasher{inner: blake3.New()}
}

// Update feeds data into the hasher.
func (hh *Hasher) Update(data []byte) {
	hh.inner.Write(data)
}

// Finalize returns the completed CardHash.
func (hh *Hasher) Finalize() CardHash {
	var digest [hashSize]byte
	sum := hh.inner.Sum(nil)
	copy(digest[:], sum)
	return CardHash{b: digest}
}
