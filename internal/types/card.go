package types

import (
	"encoding/binary"
	"path/filepath"
	"strings"
)

// CardType indicates whether a card is basic (Q&A) or cloze deletion.
type CardType int

const (
	CardTypeBasic CardType = iota
	CardTypeCloze
)

// CardContent holds the actual content of a card.
// For basic cards, Question and Answer are populated.
// For cloze cards, Text, Start, and End are populated.
type CardContent struct {
	kind     CardType
	Question string
	Answer   string
	Text     string
	Start    int
	End      int
}

// NewBasicContent creates a CardContent for a basic Q&A card.
// Leading and trailing whitespace is trimmed from both fields.
func NewBasicContent(question, answer string) CardContent {
	return CardContent{
		kind:     CardTypeBasic,
		Question: strings.TrimSpace(question),
		Answer:   strings.TrimSpace(answer),
	}
}

// NewClozeContent creates a CardContent for a cloze deletion card.
// Start and End are byte positions of the tested content inside the raw Text
// (i.e. the content between "{{" and "}}").
func NewClozeContent(text string, start, end int) CardContent {
	return CardContent{
		kind:  CardTypeCloze,
		Text:  text,
		Start: start,
		End:   end,
	}
}

// Kind returns whether this is a basic or cloze card.
func (c CardContent) Kind() CardType {
	return c.kind
}

// Hash computes the unique hash identifying this card's content.
// The algorithm matches the Rust implementation exactly, including
// start/end positions encoded as 8-byte little-endian integers.
func (c CardContent) Hash() CardHash {
	h := NewHasher()
	switch c.kind {
	case CardTypeBasic:
		h.Update([]byte("Basic"))
		h.Update([]byte(c.Question))
		h.Update([]byte(c.Answer))
	case CardTypeCloze:
		startBuf := make([]byte, 8)
		endBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(startBuf, uint64(c.Start))
		binary.LittleEndian.PutUint64(endBuf, uint64(c.End))
		h.Update([]byte("Cloze"))
		h.Update([]byte(c.Text))
		h.Update(startBuf)
		h.Update(endBuf)
	}
	return h.Finalize()
}

// FamilyHash returns the shared hash for all cloze cards derived from the
// same text. Returns nil for basic cards.
func (c CardContent) FamilyHash() *CardHash {
	if c.kind != CardTypeCloze {
		return nil
	}
	h := NewHasher()
	h.Update([]byte("Cloze"))
	h.Update([]byte(c.Text))
	hash := h.Finalize()
	return &hash
}

// Card is a flashcard with content, source location metadata, and a cached hash.
type Card struct {
	deckName  DeckName
	filePath  string
	lineStart int
	lineEnd   int
	content   CardContent
	hash      CardHash
}

// NewCard creates a Card. The content hash is computed immediately on creation.
func NewCard(
	deckName DeckName,
	filePath string,
	lineStart, lineEnd int,
	content CardContent,
) Card {
	return Card{
		deckName:  deckName,
		filePath:  filePath,
		lineStart: lineStart,
		lineEnd:   lineEnd,
		content:   content,
		hash:      content.Hash(),
	}
}

// DeckName returns the name of the deck this card belongs to.
func (c Card) DeckName() DeckName {
	return c.deckName
}

// FilePath returns the absolute path of the source file.
func (c Card) FilePath() string {
	return c.filePath
}

// LineStart returns the first line number of the card in its source file.
func (c Card) LineStart() int {
	return c.lineStart
}

// LineEnd returns the last line number of the card in its source file.
func (c Card) LineEnd() int {
	return c.lineEnd
}

// Content returns the card's content.
func (c Card) Content() CardContent {
	return c.content
}

// Hash returns the precomputed content hash.
func (c Card) Hash() CardHash {
	return c.hash
}

// FamilyHash returns the family hash for cloze cards, or nil for basic cards.
func (c Card) FamilyHash() *CardHash {
	return c.content.FamilyHash()
}

// CardType returns whether this is a basic or cloze card.
func (c Card) CardType() CardType {
	return c.content.Kind()
}

// RelativeFilePath returns the card's source file path relative to the given
// collection root directory.
func (c Card) RelativeFilePath(collectionRoot string) (string, error) {
	return filepath.Rel(collectionRoot, c.filePath)
}

// HTMLFront and HTMLBack are implemented in the internal/markdown package as
// standalone functions to avoid a circular import between types and markdown.
// Use markdown.HTMLFront(card, deckFilePath) and markdown.HTMLBack(card, deckFilePath).
