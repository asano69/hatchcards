// Package fsrs implements the Free Spaced Repetition Scheduler (FSRS) algorithm
// and the Grade type used throughout the application.
package fsrs

import (
	"encoding/json"
	"math"
	"time"

	"github.com/asano69/hashcards/internal/errs"
	"github.com/asano69/hashcards/internal/types"
)

// W contains the 19 FSRS model weights.
var W = [19]float64{
	0.40255, 1.18385, 3.173, 15.69105, 7.1949, 0.5345, 1.4604, 0.0046, 1.54575, 0.1192,
	1.01925, 1.9395, 0.11, 0.29605, 2.2698, 0.2315, 2.9898, 0.51655, 0.6621,
}

// FSRS formula constants.
const (
	fsrsF = 19.0 / 81.0
	fsrsC = -0.5

	targetRecall = 0.9
	minInterval  = 1.0
	maxInterval  = 256.0
)

// Grade represents a user's self-reported recall quality after reviewing a card.
type Grade int

const (
	GradeForgot Grade = iota // 1 in FSRS formulas
	GradeHard                // 2
	GradeGood                // 3
	GradeEasy                // 4
)

// Float64 converts a Grade to the numeric value used in FSRS formulas
// (Forgot=1, Hard=2, Good=3, Easy=4).
func (g Grade) Float64() float64 {
	return float64(g) + 1.0
}

// String returns the lowercase string used to store the grade in SQLite.
// It also implements fmt.Stringer.
func (g Grade) String() string {
	switch g {
	case GradeForgot:
		return "forgot"
	case GradeHard:
		return "hard"
	case GradeGood:
		return "good"
	case GradeEasy:
		return "easy"
	default:
		return "unknown"
	}
}

// jsonName returns the capitalized name used in JSON output.
// This matches Rust's serde default, which serializes enum variants
// using their declared name (PascalCase).
func (g Grade) jsonName() string {
	switch g {
	case GradeForgot:
		return "Forgot"
	case GradeHard:
		return "Hard"
	case GradeGood:
		return "Good"
	case GradeEasy:
		return "Easy"
	default:
		return "Unknown"
	}
}

// MarshalJSON serializes Grade as a capitalized JSON string
// (e.g. "Forgot", "Hard", "Good", "Easy").
func (g Grade) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.jsonName())
}

// UnmarshalJSON deserializes a Grade from a capitalized JSON string.
func (g *Grade) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseGradeJSON(s)
	if err != nil {
		return err
	}
	*g = parsed
	return nil
}

// ParseGradeJSON parses a capitalized JSON grade string.
func ParseGradeJSON(s string) (Grade, error) {
	switch s {
	case "Forgot":
		return GradeForgot, nil
	case "Hard":
		return GradeHard, nil
	case "Good":
		return GradeGood, nil
	case "Easy":
		return GradeEasy, nil
	default:
		return 0, errs.Newf("invalid grade: %q", s)
	}
}

// ParseGrade parses the lowercase DB string back into a Grade.
func ParseGrade(s string) (Grade, error) {
	switch s {
	case "forgot":
		return GradeForgot, nil
	case "hard":
		return GradeHard, nil
	case "good":
		return GradeGood, nil
	case "easy":
		return GradeEasy, nil
	default:
		return 0, errs.Newf("invalid grade string: %q", s)
	}
}

// Retrievability computes the probability of recall given elapsed time t (days)
// and current stability s (days).
func Retrievability(t, s float64) float64 {
	return math.Pow(1.0+fsrsF*(t/s), fsrsC)
}

// Interval returns the number of days to the next review that achieves
// targetRecall given stability s.
func Interval(targetRecall, s float64) float64 {
	return (s / fsrsF) * (math.Pow(targetRecall, 1.0/fsrsC) - 1.0)
}

// InitialStability returns the starting stability for a given grade on the
// very first review of a card.
func InitialStability(g Grade) float64 {
	// W[0]=Forgot, W[1]=Hard, W[2]=Good, W[3]=Easy
	return W[g]
}

// InitialDifficulty returns the starting difficulty for a given grade on the
// very first review of a card.
func InitialDifficulty(g Grade) float64 {
	return clampDifficulty(W[4] - math.Exp(W[5]*(g.Float64()-1.0)) + 1.0)
}

// NewStability computes the updated stability after a review.
func NewStability(d, s, r float64, g Grade) float64 {
	if g == GradeForgot {
		return stabilityAfterForgetting(d, s, r)
	}
	return stabilityAfterSuccess(d, s, r, g)
}

// NewDifficulty computes the updated difficulty after a review.
func NewDifficulty(d float64, g Grade) float64 {
	return clampDifficulty(W[7]*InitialDifficulty(GradeEasy) + (1.0-W[7])*difficultyPoint(d, g))
}

// stabilityAfterSuccess computes stability when the user remembered the card.
func stabilityAfterSuccess(d, s, r float64, g Grade) float64 {
	tD := 11.0 - d
	tS := math.Pow(s, -W[9])
	tR := math.Exp(W[10]*(1.0-r)) - 1.0
	h := 1.0
	if g == GradeHard {
		h = W[15]
	}
	b := 1.0
	if g == GradeEasy {
		b = W[16]
	}
	c := math.Exp(W[8])
	alpha := 1.0 + tD*tS*tR*h*b*c
	return s * alpha
}

// stabilityAfterForgetting computes stability when the user forgot the card.
func stabilityAfterForgetting(d, s, r float64) float64 {
	dF := math.Pow(d, -W[12])
	sF := math.Pow(s+1.0, W[13]) - 1.0
	rF := math.Exp(W[14] * (1.0 - r))
	result := dF * sF * rF * W[11]
	return math.Min(result, s)
}

// difficultyPoint computes the mean-reverting difficulty adjustment.
func difficultyPoint(d float64, g Grade) float64 {
	return d + deltaD(g)*((10.0-d)/9.0)
}

// deltaD returns the raw difficulty delta for a given grade.
func deltaD(g Grade) float64 {
	return -W[6] * (g.Float64() - 3.0)
}

// clampDifficulty ensures difficulty stays in the valid range [1, 10].
func clampDifficulty(d float64) float64 {
	return math.Max(1.0, math.Min(10.0, d))
}

// UpdatePerformance computes new scheduling parameters after a review.
// It is the Go equivalent of update_performance() in the Rust
// implementation (src/types/performance.rs).
func UpdatePerformance(
	perf types.Performance,
	grade Grade,
	reviewedAt types.Timestamp,
) types.ReviewedPerformance {
	var stability, difficulty float64
	var reviewCount int

	today := reviewedAt.Date().Time()

	if perf.IsNew() {
		stability = InitialStability(grade)
		difficulty = InitialDifficulty(grade)
		reviewCount = 0
	} else {
		rp := perf.Reviewed()
		lastDate := rp.LastReviewedAt.Date().Time()
		// Compute elapsed days as a float, matching Rust's num_days() cast.
		elapsedDays := float64(int64(today.Sub(lastDate) / (24 * time.Hour)))
		recall := Retrievability(elapsedDays, rp.Stability)
		stability = NewStability(rp.Difficulty, rp.Stability, recall, grade)
		difficulty = NewDifficulty(rp.Difficulty, grade)
		reviewCount = rp.ReviewCount
	}

	intervalRaw := Interval(targetRecall, stability)
	intervalRounded := math.Round(intervalRaw)
	intervalClamped := math.Max(minInterval, math.Min(maxInterval, intervalRounded))
	intervalDays := int64(intervalClamped)
	dueDate := types.NewDate(today.Add(time.Duration(intervalDays) * 24 * time.Hour))

	return types.ReviewedPerformance{
		LastReviewedAt: reviewedAt,
		Stability:      stability,
		Difficulty:     difficulty,
		IntervalRaw:    intervalRaw,
		IntervalDays:   intervalDays,
		DueDate:        dueDate,
		ReviewCount:    reviewCount + 1,
	}
}
