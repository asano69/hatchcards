// Package parser reads JSON deck files and extracts flashcards.
//
// # Deck file format
//
// A deck file is a JSON array of card entries. Each entry must set its own
// "deckName" field. The filename itself is arbitrary and carries no
// meaning — a single file may freely mix cards belonging to different decks.
//
// Two card types are supported:
//
// Basic card — "kind": "basic", with "question" and "answer" fields:
//
//	{"kind": "basic", "question": "What is the capital of France?", "answer": "Paris.", "deckName": "Geography"}
//
// Cloze card — "kind": "cloze", with a "text" field containing one or more
// [deletion] spans:
//
//	{"kind": "cloze", "text": "The capital of [France] is [Paris].", "deckName": "Geography"}
//
// "question", "answer", and "text" hold Markdown source. Newlines and other
// special characters are represented using standard JSON string escapes
// (e.g. "\n"), which encoding/json decodes automatically — no additional
// escaping mechanism is needed.
package parser

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/asano69/hatchards/internal/errs"
	"github.com/asano69/hatchards/internal/types"
)

// jsonCard is the on-disk representation of a single entry in a deck's
// JSON array.
type jsonCard struct {
	DeckName string         `json:"deckName"`
	Kind     types.CardType `json:"kind"`
	Question string         `json:"question,omitempty"`
	Answer   string         `json:"answer,omitempty"`
	Text     string         `json:"text,omitempty"`
}

// ParseFile reads the JSON deck file at path and returns all cards it
// contains. Each entry's deck name comes from its own required "deck_name"
// field; the filename plays no role in naming.
func ParseFile(path string) ([]types.Card, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Newf("open deck file %s: %v", path, err)
	}

	var entries []jsonCard
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, errs.Newf("parse deck file %s: %v", path, err)
	}

	var cards []types.Card
	for i, entry := range entries {
		// entryNo is the 1-based position of this entry in the JSON array.
		// It stands in for a line number in error messages and card
		// ordering, since JSON entries don't map 1:1 to source lines.
		entryNo := i + 1

		if entry.DeckName == "" {
			return nil, errs.Newf("%s: entry %d: missing required field \"deckName\"", path, entryNo)
		}

		switch entry.Kind {
		case types.CardTypeBasic:
			content := types.NewBasicContent(entry.Question, entry.Answer)
			cards = append(cards, types.NewCard(entry.DeckName, path, entryNo, entryNo, content))
		case types.CardTypeCloze:
			cleanText, spans := extractClozeSpans(entry.Text)
			for _, sp := range spans {
				content := types.NewClozeContent(cleanText, sp.start, sp.end)
				cards = append(cards, types.NewCard(entry.DeckName, path, entryNo, entryNo, content))
			}
		default:
			return nil, errs.Newf("%s: entry %d: unknown card kind %q", path, entryNo, entry.Kind)
		}
	}
	// Deduplicate cards by hash within this file.
	seen := make(map[types.CardHash]struct{}, len(cards))
	unique := cards[:0]
	for _, c := range cards {
		if _, exists := seen[c.Hash()]; !exists {
			seen[c.Hash()] = struct{}{}
			unique = append(unique, c)
		}
	}
	return unique, nil
}

// -------------------------------------------------------------------------
// Cloze span extraction
// -------------------------------------------------------------------------

type span struct{ start, end int }

// extractClozeSpans parses cloze text containing [deletion] markers.
// It returns the clean text (without brackets) and the byte positions of each
// deletion within that clean text.
//
// Special cases handled:
//   - ![ ... ] — Markdown image syntax: brackets are kept in clean text
//   - \[ and \] — escaped brackets: treated as literal text, not cloze markers
func extractClozeSpans(raw string) (string, []span) {
	raw = strings.TrimSpace(raw)
	bytes := []byte(raw)
	var cleanBytes []byte
	var spans []span

	imageMode := false
	escapeMode := false
	var startIdx *int // byte position in cleanBytes where current deletion starts

	for i := 0; i < len(bytes); i++ {
		c := bytes[i]
		switch c {
		case '[':
			if imageMode {
				cleanBytes = append(cleanBytes, c)
			} else if escapeMode {
				escapeMode = false
				cleanBytes = append(cleanBytes, c)
			} else {
				// Start of a cloze deletion.
				idx := len(cleanBytes)
				startIdx = &idx
			}
		case ']':
			if imageMode {
				imageMode = false
				cleanBytes = append(cleanBytes, c)
			} else if escapeMode {
				escapeMode = false
				cleanBytes = append(cleanBytes, c)
			} else if startIdx != nil {
				// End of a cloze deletion.
				end := len(cleanBytes) - 1
				spans = append(spans, span{start: *startIdx, end: end})
				startIdx = nil
			}
		case '!':
			// image_mode is set only when '!' is immediately before '['.
			if !imageMode && i+1 < len(bytes) && bytes[i+1] == '[' {
				imageMode = true
			}
			cleanBytes = append(cleanBytes, c)
		case '\\':
			// escape_mode is set only when '\' is immediately before '[' or ']'.
			if !escapeMode && i+1 < len(bytes) && (bytes[i+1] == '[' || bytes[i+1] == ']') {
				escapeMode = true
			} else {
				cleanBytes = append(cleanBytes, c)
			}
		default:
			cleanBytes = append(cleanBytes, c)
		}
	}

	return string(cleanBytes), spans
}
