// Package state manages the mutable state of a single drill session: the
// ordered queue of cards still to review, the list of completed reviews, and
// the session start time.
//
// State is NOT safe for concurrent use. The drill server serialises all
// access through a single mutex held at the handler level.
package state

import (
	"github.com/asano69/hashcards/internal/collection"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/rng"
	"github.com/asano69/hashcards/internal/types"
)

// CompletedReview records the outcome of reviewing one card.
type CompletedReview struct {
	Card        types.Card
	Grade       fsrs.Grade
	ReviewedAt  types.Timestamp
	Performance types.ReviewedPerformance
	// OriginalPerformance is the performance before this review, used by Undo.
	OriginalPerformance types.Performance
}

// State holds the full mutable state of an active drill session.
type State struct {
	// StartedAt is the time the session was created.
	StartedAt types.Timestamp
	// Queue is the ordered list of cards not yet reviewed this session.
	Queue []collection.DueCard
	// Done is the list of reviews completed so far, in order.
	Done []CompletedReview
	// Revealed tracks whether the answer for the current card is showing.
	Revealed bool
	// Finished is set when all cards have been graded and the session is saved.
	Finished bool
}

// New creates a State from a shuffled due-card list.
func New(due []collection.DueCard, r *rng.TinyRng) *State {
	shuffled := rng.Shuffle(due, r)
	return &State{
		StartedAt: types.Now(),
		Queue:     shuffled,
	}
}

// Current returns the card at the front of the queue and whether the queue
// is non-empty.
func (s *State) Current() (collection.DueCard, bool) {
	if len(s.Queue) == 0 {
		return collection.DueCard{}, false
	}
	return s.Queue[0], true
}

// IsFinished returns true when the session is complete.
func (s *State) IsFinished() bool {
	return s.Finished
}

// Remaining returns the number of cards still in the queue.
func (s *State) Remaining() int {
	return len(s.Queue)
}

// Total returns the total number of cards in the session (reviewed + queued).
func (s *State) Total() int {
	return len(s.Done) + len(s.Queue)
}

// ReviewedCount returns the number of cards reviewed so far.
func (s *State) ReviewedCount() int {
	return len(s.Done)
}

// Reveal flips the current card to show the answer.
func (s *State) Reveal() {
	s.Revealed = true
}

// shouldRepeat returns true when the grade warrants re-queuing the card
// (Forgot or Hard), matching the Rust implementation's should_repeat().
func shouldRepeat(grade fsrs.Grade) bool {
	return grade == fsrs.GradeForgot || grade == fsrs.GradeHard
}

// Grade records a grade for the current card and advances the queue.
// It returns the updated performance and whether the operation succeeded.
// The queue must be non-empty and the card must have been revealed.
func (s *State) Grade(grade fsrs.Grade) (types.ReviewedPerformance, bool) {
	if len(s.Queue) == 0 {
		return types.ReviewedPerformance{}, false
	}

	dc := s.Queue[0]
	s.Queue = s.Queue[1:]
	s.Revealed = false

	now := types.Now()
	newPerf := fsrs.UpdatePerformance(dc.Performance, grade, now)

	s.Done = append(s.Done, CompletedReview{
		Card:                dc.Card,
		Grade:               grade,
		ReviewedAt:          now,
		Performance:         newPerf,
		OriginalPerformance: dc.Performance,
	})

	// Re-queue at the end so the user sees the card again this session.
	if shouldRepeat(grade) {
		s.Queue = append(s.Queue, dc)
	}

	return newPerf, true
}

// Undo reverses the most recent grade, putting the card back at the front of
// the queue. It returns false when there is nothing to undo.
func (s *State) Undo() bool {
	if len(s.Done) == 0 {
		return false
	}
	last := s.Done[len(s.Done)-1]
	s.Done = s.Done[:len(s.Done)-1]

	// If the graded card was re-queued (Forgot/Hard), remove it from the tail.
	if shouldRepeat(last.Grade) {
		s.Queue = s.Queue[:len(s.Queue)-1]
	}

	// Restore the card to the front of the queue with its original performance.
	s.Queue = append([]collection.DueCard{{
		Card:        last.Card,
		Performance: last.OriginalPerformance,
	}}, s.Queue...)
	s.Revealed = false
	s.Finished = false
	return true
}

// Finish marks the session as complete.
func (s *State) Finish() {
	s.Finished = true
}

// ToReviewRecords converts the completed reviews into db.ReviewRecord values
// ready to be persisted via db.SaveSession.
func (s *State) ToReviewRecords() []db.ReviewRecord {
	records := make([]db.ReviewRecord, 0, len(s.Done))
	for _, cr := range s.Done {
		records = append(records, db.ReviewRecord{
			CardHash:     cr.Card.Hash(),
			ReviewedAt:   cr.ReviewedAt,
			Grade:        cr.Grade,
			Stability:    cr.Performance.Stability,
			Difficulty:   cr.Performance.Difficulty,
			IntervalRaw:  cr.Performance.IntervalRaw,
			IntervalDays: cr.Performance.IntervalDays,
			DueDate:      cr.Performance.DueDate,
		})
	}
	return records
}
