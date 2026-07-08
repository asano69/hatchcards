package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/asano69/hatchards/internal/types"
)

// writeDeck creates a temporary .json file with the given deck name stem and
// returns its path. The file is cleaned up automatically via t.Cleanup.
func writeDeck(t *testing.T, stem, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, stem+".json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeDeck: %v", err)
	}
	return path
}

// parseCards marshals entries to JSON, writes it as a deck file, and parses
// it via ParseFile, failing the test on error.
func parseCards(t *testing.T, entries []jsonCard) []types.Card {
	t.Helper()
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal entries: %v", err)
	}
	path := writeDeck(t, "test_deck", string(data))
	cards, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return cards
}

// assertBasic checks that cards[idx] is a basic card with the given question and answer.
func assertBasic(t *testing.T, cards []types.Card, idx int, question, answer string) {
	t.Helper()
	if idx >= len(cards) {
		t.Fatalf("index %d out of range (len=%d)", idx, len(cards))
	}
	cc := cards[idx].Content()
	if cc.Kind() != types.CardTypeBasic {
		t.Errorf("cards[%d]: expected basic card, got cloze", idx)
		return
	}
	if cc.Question != question {
		t.Errorf("cards[%d].Question = %q, want %q", idx, cc.Question, question)
	}
	if cc.Answer != answer {
		t.Errorf("cards[%d].Answer = %q, want %q", idx, cc.Answer, answer)
	}
}

// assertCloze checks that cards[idx] is a cloze card with the given text, start, end.
func assertCloze(t *testing.T, cards []types.Card, idx int, text string, start, end int) {
	t.Helper()
	if idx >= len(cards) {
		t.Fatalf("index %d out of range (len=%d)", idx, len(cards))
	}
	cc := cards[idx].Content()
	if cc.Kind() != types.CardTypeCloze {
		t.Errorf("cards[%d]: expected cloze card, got basic", idx)
		return
	}
	if cc.Text != text {
		t.Errorf("cards[%d].Text = %q, want %q", idx, cc.Text, text)
	}
	if cc.Start != start {
		t.Errorf("cards[%d].Start = %d, want %d", idx, cc.Start, start)
	}
	if cc.End != end {
		t.Errorf("cards[%d].End = %d, want %d", idx, cc.End, end)
	}
}

// ---- Basic card tests ----

func TestEmptyArray(t *testing.T) {
	cards := parseCards(t, []jsonCard{})
	if len(cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(cards))
	}
}

func TestBasicCard(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "basic", Question: "What is Rust?", Answer: "A systems programming language.", DeckName: "Test"},
	})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	assertBasic(t, cards, 0, "What is Rust?", "A systems programming language.")
}

func TestMultilineQA(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "basic", Question: "foo\nbaz\nbaz", Answer: "FOO\nBAR\nBAZ", DeckName: "Test"},
	})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	assertBasic(t, cards, 0, "foo\nbaz\nbaz", "FOO\nBAR\nBAZ")
}

func TestTwoQuestions(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "basic", Question: "foo", Answer: "bar", DeckName: "Test"},
		{Kind: "basic", Question: "baz", Answer: "quux", DeckName: "Test"},
	})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	assertBasic(t, cards, 0, "foo", "bar")
	assertBasic(t, cards, 1, "baz", "quux")
}

// ---- Cloze card tests ----

func TestClozeFollowedByQuestion(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "cloze", Text: "[foo]", DeckName: "Test"},
		{Kind: "basic", Question: "Question", Answer: "Answer", DeckName: "Test"},
	})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "foo", 0, 2)
	assertBasic(t, cards, 1, "Question", "Answer")
}

func TestClozeSingle(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "Foo [bar] baz.", DeckName: "Test"}})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "Foo bar baz.", 4, 6)
}

func TestClozeMultiple(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "Foo [bar] baz [quux].", DeckName: "Test"}})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "Foo bar baz quux.", 4, 6)
	assertCloze(t, cards, 1, "Foo bar baz quux.", 12, 15)
}

func TestClozeWithImage(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "Foo [bar] ![](image.jpg) [quux].", DeckName: "Test"}})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "Foo bar ![](image.jpg) quux.", 4, 6)
	assertCloze(t, cards, 1, "Foo bar ![](image.jpg) quux.", 23, 26)
}

func TestClozeWithEscapedSquareBracket(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "Key: [`\\[`]", DeckName: "Test"}})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "Key: `[`", 5, 7)
}

func TestClozeWithMultipleEscapedBrackets(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "\\[markdown\\] [`\\[cloze\\]`]", DeckName: "Test"}})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "[markdown] `[cloze]`", 11, 19)
}

func TestMultiLineCloze(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "[foo]\n[bar]\nbaz.", DeckName: "Test"}})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "foo\nbar\nbaz.", 0, 2)
	assertCloze(t, cards, 1, "foo\nbar\nbaz.", 4, 6)
}

func TestTwoClozes(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "cloze", Text: "[foo]", DeckName: "Test"},
		{Kind: "cloze", Text: "[bar]", DeckName: "Test"},
	})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	assertCloze(t, cards, 0, "foo", 0, 2)
	assertCloze(t, cards, 1, "bar", 0, 2)
}

func TestClozeWithInitialBlankLine(t *testing.T) {
	text := "\nBuild something people want in Lisp.\n\n\u2014 [Paul Graham], [_Hackers and Painters_]\n\n"
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: text, DeckName: "Test"}})
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	wantText := "Build something people want in Lisp.\n\n\u2014 Paul Graham, _Hackers and Painters_"
	assertCloze(t, cards, 0, wantText, 42, 52)
	assertCloze(t, cards, 1, wantText, 55, 76)
}

// ---- Deduplication tests ----

func TestIdenticalBasicCards(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "basic", Question: "foo", Answer: "bar", DeckName: "Test"},
		{Kind: "basic", Question: "foo", Answer: "bar", DeckName: "Test"},
	})
	if len(cards) != 1 {
		t.Errorf("expected 1 card after dedup, got %d", len(cards))
	}
}

func TestIdenticalClozeCards(t *testing.T) {
	cards := parseCards(t, []jsonCard{
		{Kind: "cloze", Text: "foo [bar]", DeckName: "Test"},
		{Kind: "cloze", Text: "foo [bar]", DeckName: "Test"},
	})
	if len(cards) != 1 {
		t.Errorf("expected 1 card after dedup, got %d", len(cards))
	}
}

func TestIdenticalCardsAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	data, err := json.Marshal([]jsonCard{{Kind: "basic", Question: "foo", Answer: "bar", DeckName: "Test"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"file1.json", "file2.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	cards1, err := ParseFile(filepath.Join(dir, "file1.json"))
	if err != nil {
		t.Fatal(err)
	}
	cards2, err := ParseFile(filepath.Join(dir, "file2.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cards1) != 1 || len(cards2) != 1 {
		t.Fatalf("expected 1 card from each file, got %d and %d", len(cards1), len(cards2))
	}
	if !cards1[0].Hash().Equal(cards2[0].Hash()) {
		t.Error("same card content in two files should produce the same hash")
	}
}

// ---- Deck name tests ----

// TestDeckNameFromJSON verifies that a card's deck name comes from its
// "deckName" field, independent of the file's name.
func TestDeckNameFromJSON(t *testing.T) {
	path := writeDeck(t, "SomeFile", `[{"kind":"basic","question":"foo","answer":"bar","deckName":"MyDeck"}]`)
	cards, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	if cards[0].DeckName() != "MyDeck" {
		t.Errorf("DeckName = %q, want %q", cards[0].DeckName(), "MyDeck")
	}
}

// TestMissingDeckName verifies that an entry without "deckName" is rejected.
func TestMissingDeckName(t *testing.T) {
	path := writeDeck(t, "bad_deck", `[{"kind":"basic","question":"foo","answer":"bar"}]`)
	if _, err := ParseFile(path); err == nil {
		t.Error("expected error for missing deckName, got nil")
	}
}

// ---- Special character tests ----

func TestClozeDeletionWithExclamationSign(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "The notation [$n!$] means 'n factorial'.", DeckName: "Test"}})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	cc := cards[0].Content()
	if cc.Kind() != types.CardTypeCloze {
		t.Fatal("expected cloze card")
	}
	wantText := "The notation $n!$ means 'n factorial'."
	if cc.Text != wantText {
		t.Errorf("Text = %q, want %q", cc.Text, wantText)
	}
}

func TestClozeDeletionWithMath(t *testing.T) {
	cards := parseCards(t, []jsonCard{{Kind: "cloze", Text: "The string `\\alpha` renders as [$\\alpha$].", DeckName: "Test"}})
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	cc := cards[0].Content()
	wantText := "The string `\\alpha` renders as $\\alpha$."
	if cc.Text != wantText {
		t.Errorf("Text = %q, want %q", cc.Text, wantText)
	}
}

// ---- Error handling ----

func TestUnknownCardType(t *testing.T) {
	path := writeDeck(t, "bad_deck", `[{"kind":"invalid","deckName":"Test"}]`)
	if _, err := ParseFile(path); err == nil {
		t.Error("expected error for unknown card type, got nil")
	}
}

func TestInvalidJSON(t *testing.T) {
	path := writeDeck(t, "bad_deck", `not json`)
	if _, err := ParseFile(path); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
