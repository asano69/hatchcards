// Package export implements the "export" subcommand, which writes the full
// review history from the performance database to stdout as JSON.
package export

import (
	"encoding/json"
	"io"

	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/types"
)

// ExportedReview is one review record in the JSON output.
// Field names and types match the Rust serde output exactly.
type ExportedReview struct {
	CardHash     string     `json:"card_hash"`
	ReviewedAt   string     `json:"reviewed_at"`
	Grade        fsrs.Grade `json:"grade"`
	Stability    float64    `json:"stability"`
	Difficulty   float64    `json:"difficulty"`
	IntervalRaw  float64    `json:"interval_raw"`
	IntervalDays int64      `json:"interval_days"`
	DueDate      string     `json:"due_date"`
}

// ExportedSession is one session record in the JSON output.
type ExportedSession struct {
	SessionID int64            `json:"session_id"`
	StartedAt string           `json:"started_at"`
	EndedAt   string           `json:"ended_at"`
	Reviews   []ExportedReview `json:"reviews"`
}

// Run reads all sessions and reviews from the database at dbPath and writes
// them as a JSON array to out. The output is indented for readability.
func Run(dbPath string, out io.Writer) error {
	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	sessions, err := database.GetAllSessions()
	if err != nil {
		return err
	}

	exported := make([]ExportedSession, 0, len(sessions))
	for _, s := range sessions {
		reviews, err := database.GetReviewsForSession(s.SessionID)
		if err != nil {
			return err
		}

		expReviews := make([]ExportedReview, 0, len(reviews))
		for _, r := range reviews {
			expReviews = append(expReviews, exportReview(r.Data))
		}

		exported = append(exported, ExportedSession{
			SessionID: s.SessionID,
			StartedAt: s.StartedAt.String(),
			EndedAt:   s.EndedAt.String(),
			Reviews:   expReviews,
		})
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(exported)
}

// exportReview converts a db.ReviewRecord to its JSON-serialisable form.
func exportReview(r db.ReviewRecord) ExportedReview {
	return ExportedReview{
		CardHash:     r.CardHash.Hex(),
		ReviewedAt:   r.ReviewedAt.String(),
		Grade:        r.Grade,
		Stability:    r.Stability,
		Difficulty:   r.Difficulty,
		IntervalRaw:  r.IntervalRaw,
		IntervalDays: r.IntervalDays,
		DueDate:      r.DueDate.String(),
	}
}

// ExportedCard is the JSON representation of a card's current performance,
// used when exporting the full card state rather than session history.
type ExportedCard struct {
	CardHash    string               `json:"card_hash"`
	AddedAt     string               `json:"added_at,omitempty"`
	Performance *ExportedPerformance `json:"performance,omitempty"`
}

// ExportedPerformance mirrors types.ReviewedPerformance for JSON output.
type ExportedPerformance struct {
	LastReviewedAt string  `json:"last_reviewed_at"`
	Stability      float64 `json:"stability"`
	Difficulty     float64 `json:"difficulty"`
	IntervalRaw    float64 `json:"interval_raw"`
	IntervalDays   int64   `json:"interval_days"`
	DueDate        string  `json:"due_date"`
	ReviewCount    int     `json:"review_count"`
}

// exportPerformance converts a types.Performance to its JSON form.
// Returns nil for new (never reviewed) cards.
func exportPerformance(p types.Performance) *ExportedPerformance {
	rp := p.Reviewed()
	if rp == nil {
		return nil
	}
	return &ExportedPerformance{
		LastReviewedAt: rp.LastReviewedAt.String(),
		Stability:      rp.Stability,
		Difficulty:     rp.Difficulty,
		IntervalRaw:    rp.IntervalRaw,
		IntervalDays:   rp.IntervalDays,
		DueDate:        rp.DueDate.String(),
		ReviewCount:    rp.ReviewCount,
	}
}
