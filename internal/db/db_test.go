package db

import (
	"testing"

	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/types"
)

// openMemory opens an in-memory database for testing.
func openMemory(t *testing.T) *Database {
	t.Helper()
	db, err := OpenScratch()
	if err != nil {
		t.Fatalf("OpenScratch: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestInsertCard matches Rust's test_insert_card.
func TestInsertCard(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}

	hashes, err := db.CardHashes()
	if err != nil {
		t.Fatalf("CardHashes: %v", err)
	}
	if _, ok := hashes[cardHash]; !ok {
		t.Error("card hash not found after insert")
	}

	perf, err := db.GetCardPerformance(cardHash)
	if err != nil {
		t.Fatalf("GetCardPerformance: %v", err)
	}
	if !perf.IsNew() {
		t.Error("expected new card performance after first insert")
	}

	dueToday, err := db.DueToday(now.Date())
	if err != nil {
		t.Fatalf("DueToday: %v", err)
	}
	if _, ok := dueToday[cardHash]; !ok {
		t.Error("new card should be due today")
	}
}

func TestPocketBaseCollectionsExist(t *testing.T) {
	db := openMemory(t)

	for _, name := range []string{"cards", "sessions", "reviews"} {
		if _, err := db.App().FindCollectionByNameOrId(name); err != nil {
			t.Fatalf("FindCollectionByNameOrId(%q): %v", name, err)
		}
	}
}

// TestInsertTwice matches Rust's test_insert_twice.
func TestInsertTwice(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("first InsertCard: %v", err)
	}
	if err := db.InsertCard(cardHash, now); err == nil {
		t.Error("second InsertCard: expected error, got nil")
	}
}

// TestUpdatePerformance matches Rust's test_update_performance.
func TestUpdatePerformance(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}

	rp := types.ReviewedPerformance{
		LastReviewedAt: now,
		Stability:      2.0,
		Difficulty:     2.0,
		IntervalRaw:    1.0,
		IntervalDays:   1,
		DueDate:        now.Date(),
		ReviewCount:    1,
	}
	perf := types.ReviewedCardPerformance(rp)

	if err := db.UpdateCardPerformance(cardHash, perf); err != nil {
		t.Fatalf("UpdateCardPerformance: %v", err)
	}

	fetched, err := db.GetCardPerformance(cardHash)
	if err != nil {
		t.Fatalf("GetCardPerformance: %v", err)
	}
	if fetched.IsNew() {
		t.Fatal("expected reviewed performance, got new")
	}
	frp := fetched.Reviewed()
	if !frp.LastReviewedAt.Equal(now) {
		t.Errorf("LastReviewedAt: got %v, want %v", frp.LastReviewedAt, now)
	}
	if frp.Stability != rp.Stability {
		t.Errorf("Stability: got %f, want %f", frp.Stability, rp.Stability)
	}
	if frp.Difficulty != rp.Difficulty {
		t.Errorf("Difficulty: got %f, want %f", frp.Difficulty, rp.Difficulty)
	}
	if frp.IntervalRaw != rp.IntervalRaw {
		t.Errorf("IntervalRaw: got %f, want %f", frp.IntervalRaw, rp.IntervalRaw)
	}
	if frp.IntervalDays != rp.IntervalDays {
		t.Errorf("IntervalDays: got %d, want %d", frp.IntervalDays, rp.IntervalDays)
	}
	if !frp.DueDate.Equal(rp.DueDate) {
		t.Errorf("DueDate: got %v, want %v", frp.DueDate, rp.DueDate)
	}
	if frp.ReviewCount != rp.ReviewCount {
		t.Errorf("ReviewCount: got %d, want %d", frp.ReviewCount, rp.ReviewCount)
	}

	// Verified card is still due today (DueDate == today).
	dueToday, err := db.DueToday(now.Date())
	if err != nil {
		t.Fatalf("DueToday: %v", err)
	}
	if _, ok := dueToday[cardHash]; !ok {
		t.Error("card with due date == today should appear in DueToday")
	}
}

// TestGetPerformanceNonexistent matches Rust's test_get_performance_nonexistent.
func TestGetPerformanceNonexistent(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))

	if _, err := db.GetCardPerformance(cardHash); err == nil {
		t.Error("expected error for nonexistent card, got nil")
	}
}

// TestUpdatePerformanceNonexistent matches Rust's test_update_performance_nonexistent.
func TestUpdatePerformanceNonexistent(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	perf := types.NewCardPerformance()

	if err := db.UpdateCardPerformance(cardHash, perf); err == nil {
		t.Error("expected error when updating nonexistent card, got nil")
	}
}

// TestSaveSession matches Rust's test_save_session.
func TestSaveSession(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}

	review := ReviewRecord{
		CardHash:     cardHash,
		ReviewedAt:   now,
		Grade:        fsrs.GradeGood,
		Stability:    2.0,
		Difficulty:   2.0,
		IntervalRaw:  1.0,
		IntervalDays: 1,
		DueDate:      now.Date(),
	}
	if err := db.SaveSession(now, now, []ReviewRecord{review}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	sessions, err := db.GetAllSessions()
	if err != nil {
		t.Fatalf("GetAllSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if !sessions[0].StartedAt.Equal(now) {
		t.Errorf("StartedAt: got %v, want %v", sessions[0].StartedAt, now)
	}
	if !sessions[0].EndedAt.Equal(now) {
		t.Errorf("EndedAt: got %v, want %v", sessions[0].EndedAt, now)
	}

	reviews, err := db.GetReviewsForSession(sessions[0].SessionID)
	if err != nil {
		t.Fatalf("GetReviewsForSession: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
	r := reviews[0].Data
	if !r.CardHash.Equal(cardHash) {
		t.Errorf("CardHash mismatch: got %s, want %s", r.CardHash.Hex(), cardHash.Hex())
	}
	if !r.ReviewedAt.Equal(now) {
		t.Errorf("ReviewedAt: got %v, want %v", r.ReviewedAt, now)
	}
	if r.Grade != fsrs.GradeGood {
		t.Errorf("Grade: got %v, want Good", r.Grade)
	}
	if r.Stability != 2.0 {
		t.Errorf("Stability: got %f, want 2.0", r.Stability)
	}
	if r.IntervalDays != 1 {
		t.Errorf("IntervalDays: got %d, want 1", r.IntervalDays)
	}
}

// TestDeleteNonexistentCard matches Rust's test_delete_nonexistent_card.
func TestDeleteNonexistentCard(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))

	if err := db.DeleteCard(cardHash); err == nil {
		t.Error("expected error when deleting nonexistent card, got nil")
	}
}

// TestDeleteCard matches Rust's test_delete_card.
func TestDeleteCard(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}
	if err := db.DeleteCard(cardHash); err != nil {
		t.Fatalf("DeleteCard: %v", err)
	}
	if _, err := db.GetCardPerformance(cardHash); err == nil {
		t.Error("expected error after deletion, got nil")
	}
}

// TestCountReviewsInDate verifies the review counter for a given date.
func TestCountReviewsInDate(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}

	// No reviews yet.
	count, err := db.CountReviewsInDate(now.Date())
	if err != nil {
		t.Fatalf("CountReviewsInDate: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 reviews before session, got %d", count)
	}

	review := ReviewRecord{
		CardHash:     cardHash,
		ReviewedAt:   now,
		Grade:        fsrs.GradeGood,
		Stability:    1.0,
		Difficulty:   1.0,
		IntervalRaw:  1.0,
		IntervalDays: 1,
		DueDate:      now.Date(),
	}
	if err := db.SaveSession(now, now, []ReviewRecord{review}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	count, err = db.CountReviewsInDate(now.Date())
	if err != nil {
		t.Fatalf("CountReviewsInDate: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 review after session, got %d", count)
	}
}

// TestDueTodayFutureCard verifies that a card with a future due date is NOT due today.
func TestDueTodayFutureCard(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}

	// Update with a due date far in the future.
	futureDate := types.NewDate(now.Time().AddDate(0, 0, 30))
	rp := types.ReviewedPerformance{
		LastReviewedAt: now,
		Stability:      10.0,
		Difficulty:     5.0,
		IntervalRaw:    30.0,
		IntervalDays:   30,
		DueDate:        futureDate,
		ReviewCount:    1,
	}
	if err := db.UpdateCardPerformance(cardHash, types.ReviewedCardPerformance(rp)); err != nil {
		t.Fatalf("UpdateCardPerformance: %v", err)
	}

	dueToday, err := db.DueToday(now.Date())
	if err != nil {
		t.Fatalf("DueToday: %v", err)
	}
	if _, ok := dueToday[cardHash]; ok {
		t.Error("card with future due date should NOT appear in DueToday")
	}
}

// TestReviewCreateHookSyncsCardPerformance verifies that inserting a review
// record (via SaveSession) automatically updates the matching card's cached
// performance fields, without an explicit UpdateCardPerformance call. This
// covers stage 1 of the reviews->cards hooks migration.
func TestReviewCreateHookSyncsCardPerformance(t *testing.T) {
	db := openMemory(t)
	cardHash := types.HashBytes([]byte("a"))
	now := types.Now()

	if err := db.InsertCard(cardHash, now); err != nil {
		t.Fatalf("InsertCard: %v", err)
	}

	review := ReviewRecord{
		CardHash:     cardHash,
		ReviewedAt:   now,
		Grade:        fsrs.GradeGood,
		Stability:    3.17,
		Difficulty:   5.28,
		IntervalRaw:  3.17,
		IntervalDays: 3,
		DueDate:      now.Date(),
	}
	if err := db.SaveSession(now, now, []ReviewRecord{review}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	perf, err := db.GetCardPerformance(cardHash)
	if err != nil {
		t.Fatalf("GetCardPerformance: %v", err)
	}
	if perf.IsNew() {
		t.Fatal("expected reviewed performance after review insert, got new")
	}
	rp := perf.Reviewed()
	if rp.Stability != review.Stability {
		t.Errorf("Stability: got %f, want %f", rp.Stability, review.Stability)
	}
	if rp.Difficulty != review.Difficulty {
		t.Errorf("Difficulty: got %f, want %f", rp.Difficulty, review.Difficulty)
	}
	if rp.IntervalDays != review.IntervalDays {
		t.Errorf("IntervalDays: got %d, want %d", rp.IntervalDays, review.IntervalDays)
	}
	if rp.ReviewCount != 1 {
		t.Errorf("ReviewCount: got %d, want 1", rp.ReviewCount)
	}

	// A second review should bump review_count to 2.
	if err := db.SaveSession(now, now, []ReviewRecord{review}); err != nil {
		t.Fatalf("second SaveSession: %v", err)
	}
	perf2, err := db.GetCardPerformance(cardHash)
	if err != nil {
		t.Fatalf("GetCardPerformance (2nd): %v", err)
	}
	if perf2.Reviewed().ReviewCount != 2 {
		t.Errorf("ReviewCount after 2nd review: got %d, want 2", perf2.Reviewed().ReviewCount)
	}
}
