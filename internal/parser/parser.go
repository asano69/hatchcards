// Package parser reads Markdown deck files and extracts flashcards.
//
// # Deck file format
//
// A deck file may optionally begin with a TOML frontmatter block:
//
//	---
//	name = "My Deck"
//	---
//
// When the frontmatter block is absent the deck name is derived from the
// filename without its extension.
//
// Two card formats are supported:
//
// Basic card — question and answer prefixed with "Q:" and "A:":
//
//	Q: What is the capital of France?
//	A: Paris.
//
// Cloze card — one or more [deletion] spans, prefixed with "C:":
//
//	C: The capital of [France] is [Paris].
package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/types"
)

// -------------------------------------------------------------------------
// Frontmatter parsing
// -------------------------------------------------------------------------

// parseFrontmatter reads the optional TOML frontmatter block at the top of a
// deck file. It returns the deck name (from "name = ..." if present) and the
// remaining lines after the closing "---".
//
// Frontmatter format:
//
//	---
//	name = "My Deck"
//	---
func parseFrontmatter(lines []string) (name string, rest []string) {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", lines
	}
	// Find closing "---".
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Parse the frontmatter block for a "name" key.
			for _, line := range lines[1:i] {
				if n, ok := parseTomlString(line, "name"); ok {
					name = n
				}
			}
			return name, lines[i+1:]
		}
	}
	// No closing delimiter found; treat the whole file as content.
	return "", lines
}

// parseTomlString parses a line of the form:  key = "value"
// and returns (value, true) when the key matches. It ignores leading/trailing
// whitespace and handles both single- and double-quoted values.
func parseTomlString(line, key string) (string, bool) {
	line = strings.TrimSpace(line)
	prefix := key + " ="
	if !strings.HasPrefix(line, prefix) {
		// Also accept "key=" without a space.
		prefix2 := key + "="
		if !strings.HasPrefix(line, prefix2) {
			return "", false
		}
		line = strings.TrimSpace(line[len(prefix2):])
	} else {
		line = strings.TrimSpace(line[len(prefix):])
	}
	// Strip surrounding quotes.
	if len(line) >= 2 {
		q := line[0]
		if (q == '"' || q == '\'') && line[len(line)-1] == q {
			return line[1 : len(line)-1], true
		}
	}
	return line, true
}

// -------------------------------------------------------------------------
// Line classification
// -------------------------------------------------------------------------

type lineKind int

const (
	lineKindBlank   lineKind = iota // empty or whitespace-only
	lineKindRule                    // exactly "---"
	lineKindQPrefix                 // starts with "Q:"
	lineKindAPrefix                 // starts with "A:"
	lineKindCPrefix                 // starts with "C:"
	lineKindText                    // any other non-empty line
)

func classifyLine(raw string) (lineKind, string) {
	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return lineKindBlank, ""
	case trimmed == "---":
		return lineKindRule, "---"
	case strings.HasPrefix(trimmed, "Q:"):
		return lineKindQPrefix, strings.TrimSpace(trimmed[2:])
	case strings.HasPrefix(trimmed, "A:"):
		return lineKindAPrefix, strings.TrimSpace(trimmed[2:])
	case strings.HasPrefix(trimmed, "C:"):
		return lineKindCPrefix, strings.TrimSpace(trimmed[2:])
	default:
		return lineKindText, trimmed
	}
}

// -------------------------------------------------------------------------
// Parser state machine
// -------------------------------------------------------------------------

type parserState int

const (
	stateIdle     parserState = iota // between cards
	stateQuestion                    // accumulating question lines
	stateAnswer                      // accumulating answer lines
	stateCloze                       // accumulating cloze lines
)

type parseContext struct {
	deckName  types.DeckName
	filePath  string
	cards     []types.Card
	st        parserState
	lineNo    int
	cardStart int

	questionLines []string
	answerLines   []string
	clozeLines    []string
}

// ParseFile reads the deck file at path and returns all cards it contains.
func ParseFile(path string) ([]types.Card, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errs.Newf("open deck file %s: %v", path, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, errs.Newf("read deck file %s: %v", path, err)
	}

	// Extract frontmatter; derive deck name from filename if absent.
	name, contentLines := parseFrontmatter(lines)
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	ctx := &parseContext{
		deckName: name,
		filePath: path,
		st:       stateIdle,
	}

	for i, line := range contentLines {
		ctx.lineNo = i + 1
		ctx.step(line)
	}
	ctx.flush()

	// Deduplicate cards by hash within this file, matching Rust parser behaviour.
	seen := make(map[types.CardHash]struct{}, len(ctx.cards))
	unique := ctx.cards[:0]
	for _, c := range ctx.cards {
		if _, exists := seen[c.Hash()]; !exists {
			seen[c.Hash()] = struct{}{}
			unique = append(unique, c)
		}
	}
	return unique, nil

}

func (ctx *parseContext) step(raw string) {
	kind, text := classifyLine(raw)

	switch ctx.st {
	// ------------------------------------------------------------------
	case stateIdle:
		switch kind {
		case lineKindBlank, lineKindRule:
			// Skip.
		case lineKindQPrefix:
			ctx.cardStart = ctx.lineNo
			ctx.questionLines = []string{text}
			ctx.st = stateQuestion
		case lineKindCPrefix:
			ctx.cardStart = ctx.lineNo
			ctx.clozeLines = []string{text}
			ctx.st = stateCloze
		case lineKindText:
			// Bare text line starts a question (no Q: prefix needed for
			// compatibility with the basic format).
			ctx.cardStart = ctx.lineNo
			ctx.questionLines = []string{text}
			ctx.st = stateQuestion
		}

	// ------------------------------------------------------------------
	case stateQuestion:
		switch kind {
		case lineKindAPrefix:
			ctx.answerLines = []string{text}
			ctx.st = stateAnswer
		case lineKindQPrefix:
			// New Q: — flush question-only card, start fresh.
			ctx.emitBasic(ctx.lineNo)
			ctx.cardStart = ctx.lineNo
			ctx.questionLines = []string{text}
		case lineKindCPrefix:
			ctx.emitBasic(ctx.lineNo)
			ctx.cardStart = ctx.lineNo
			ctx.clozeLines = []string{text}
			ctx.st = stateCloze
		case lineKindRule:
			ctx.emitBasic(ctx.lineNo)
			ctx.st = stateIdle
		case lineKindBlank:
			// blank line between Q and A is fine; stay in stateQuestion
		default:
			ctx.questionLines = append(ctx.questionLines, text)
		}

	// ------------------------------------------------------------------
	case stateAnswer:
		switch kind {
		case lineKindRule:
			ctx.emitBasic(ctx.lineNo)
			ctx.st = stateIdle
		case lineKindQPrefix:
			ctx.emitBasic(ctx.lineNo)
			ctx.cardStart = ctx.lineNo
			ctx.questionLines = []string{text}
			ctx.answerLines = nil
			ctx.st = stateQuestion
		case lineKindCPrefix:
			ctx.emitBasic(ctx.lineNo)
			ctx.cardStart = ctx.lineNo
			ctx.clozeLines = []string{text}
			ctx.answerLines = nil
			ctx.st = stateCloze
		case lineKindBlank:
			ctx.answerLines = append(ctx.answerLines, "")
		default:
			ctx.answerLines = append(ctx.answerLines, text)
		}

	// ------------------------------------------------------------------
	case stateCloze:
		switch kind {
		case lineKindRule:
			ctx.emitCloze(ctx.lineNo)
			ctx.st = stateIdle
		case lineKindQPrefix:
			ctx.emitCloze(ctx.lineNo)
			ctx.cardStart = ctx.lineNo
			ctx.questionLines = []string{text}
			ctx.st = stateQuestion
		case lineKindCPrefix:
			ctx.emitCloze(ctx.lineNo)
			ctx.cardStart = ctx.lineNo
			ctx.clozeLines = []string{text}
		case lineKindBlank:
			ctx.clozeLines = append(ctx.clozeLines, "")
		default:
			ctx.clozeLines = append(ctx.clozeLines, text)
		}
	}
}

// flush finalises any in-progress card at EOF.
func (ctx *parseContext) flush() {
	switch ctx.st {
	case stateAnswer, stateQuestion:
		ctx.emitBasic(ctx.lineNo)
	case stateCloze:
		ctx.emitCloze(ctx.lineNo)
	}
}

// -------------------------------------------------------------------------
// Card emission
// -------------------------------------------------------------------------

func (ctx *parseContext) emitBasic(lineEnd int) {
	question := strings.Join(ctx.questionLines, "\n")
	answer := strings.Join(trimTrailingBlanks(ctx.answerLines), "\n")
	content := types.NewBasicContent(question, answer)
	card := types.NewCard(ctx.deckName, ctx.filePath, ctx.cardStart, lineEnd, content)
	ctx.cards = append(ctx.cards, card)
	ctx.questionLines = nil
	ctx.answerLines = nil
}

func (ctx *parseContext) emitCloze(lineEnd int) {
	fullText := strings.Join(trimTrailingBlanks(ctx.clozeLines), "\n")
	// Build clean text and collect deletion spans, matching Rust parser logic.
	cleanText, spans := extractClozeSpans(fullText)
	for _, sp := range spans {
		content := types.NewClozeContent(cleanText, sp.start, sp.end)
		card := types.NewCard(ctx.deckName, ctx.filePath, ctx.cardStart, lineEnd, content)
		ctx.cards = append(ctx.cards, card)
	}
	ctx.clozeLines = nil
}

// -------------------------------------------------------------------------
// Cloze span extraction
// -------------------------------------------------------------------------

type span struct{ start, end int }

// extractClozeSpans parses cloze text containing [deletion] markers.
// It returns the clean text (without brackets) and the byte positions of each
// deletion within that clean text, matching the Rust parser's behaviour.
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

func trimTrailingBlanks(lines []string) []string {
	end := len(lines)
	for end > 0 && lines[end-1] == "" {
		end--
	}
	return lines[:end]
}
