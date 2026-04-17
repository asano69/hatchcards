package fsrs

import (
	"math"
	"testing"
	"time"

	"github.com/asano69/hashcards/internal/types"
)

// approxEq reports whether a and b differ by less than 0.01,
// matching the Rust test helper feq().
func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

// simStep mirrors the Rust test helper's Step struct.
type simStep struct {
	t, s, d, i float64
}

func (s simStep) equal(other simStep) bool {
	return approxEq(s.t, other.t) &&
		approxEq(s.s, other.s) &&
		approxEq(s.d, other.d) &&
		approxEq(s.i, other.i)
}

// sim simulates a series of reviews, matching the Rust test helper sim().
func sim(grades []Grade) []simStep {
	if len(grades) == 0 {
		return nil
	}
	t := 0.0
	rD := 0.9
	var steps []simStep

	g := grades[0]
	grades = grades[1:]
	s := InitialStability(g)
	d := InitialDifficulty(g)
	i := math.Max(math.Round(Interval(rD, s)), 1.0)
	steps = append(steps, simStep{t, s, d, i})

	for _, g := range grades {
		t += i
		r := Retrievability(i, s)
		s = NewStability(d, s, r, g)
		d = NewDifficulty(d, g)
		i = math.Max(math.Round(Interval(rD, s)), 1.0)
		steps = append(steps, simStep{t, s, d, i})
	}
	return steps
}

func TestIntervalEqualsStability(t *testing.T) {
	samples := 100
	start := 0.1
	end := 5.0
	step := (end - start) / float64(samples-1)
	for i := 0; i < samples; i++ {
		s := start + float64(i)*step
		got := Interval(0.9, s)
		if !approxEq(got, s) {
			t.Errorf("Interval(0.9, %f) = %f, want ≈ %f", s, got, s)
		}
	}
}

func TestInitialDifficultyOfForgetting(t *testing.T) {
	got := InitialDifficulty(GradeForgot)
	if got != W[4] {
		t.Errorf("InitialDifficulty(Forgot) = %f, want %f", got, W[4])
	}
}

func TestThreeEasies(t *testing.T) {
	grades := []Grade{GradeEasy, GradeEasy, GradeEasy}
	expected := []simStep{
		{t: 0.0, s: 15.69, d: 3.22, i: 16.0},
		{t: 16.0, s: 150.28, d: 2.13, i: 150.0},
		{t: 166.0, s: 1252.22, d: 1.0, i: 1252.0},
	}
	checkSim(t, grades, expected)
}

func TestThreeGoods(t *testing.T) {
	grades := []Grade{GradeGood, GradeGood, GradeGood}
	expected := []simStep{
		{t: 0.0, s: 3.17, d: 5.28, i: 3.0},
		{t: 3.0, s: 10.73, d: 5.27, i: 11.0},
		{t: 14.0, s: 34.57, d: 5.26, i: 35.0},
	}
	checkSim(t, grades, expected)
}

func TestTwoHards(t *testing.T) {
	grades := []Grade{GradeHard, GradeHard}
	expected := []simStep{
		{t: 0.0, s: 1.18, d: 6.48, i: 1.0},
		{t: 1.0, s: 1.70, d: 7.04, i: 2.0},
	}
	checkSim(t, grades, expected)
}

func TestTwoForgots(t *testing.T) {
	grades := []Grade{GradeForgot, GradeForgot}
	expected := []simStep{
		{t: 0.0, s: 0.40, d: 7.19, i: 1.0},
		{t: 1.0, s: 0.26, d: 8.08, i: 1.0},
	}
	checkSim(t, grades, expected)
}

func TestGoodThenForgot(t *testing.T) {
	grades := []Grade{GradeGood, GradeForgot}
	expected := []simStep{
		{t: 0.0, s: 3.17, d: 5.28, i: 3.0},
		{t: 3.0, s: 1.06, d: 6.8, i: 1.0},
	}
	checkSim(t, grades, expected)
}

func checkSim(t *testing.T, grades []Grade, expected []simStep) {
	t.Helper()
	actual := sim(grades)
	if len(actual) != len(expected) {
		t.Fatalf("got %d steps, want %d", len(actual), len(expected))
	}
	for i := range expected {
		if !actual[i].equal(expected[i]) {
			t.Errorf("step[%d]: got {t:%.2f s:%.2f d:%.2f i:%.2f}, want {t:%.2f s:%.2f d:%.2f i:%.2f}",
				i,
				actual[i].t, actual[i].s, actual[i].d, actual[i].i,
				expected[i].t, expected[i].s, expected[i].d, expected[i].i)
		}
	}
}

func TestGradeString(t *testing.T) {
	tests := []struct {
		grade Grade
		want  string
	}{
		{GradeForgot, "forgot"},
		{GradeHard, "hard"},
		{GradeGood, "good"},
		{GradeEasy, "easy"},
	}
	for _, tt := range tests {
		if tt.grade.String() != tt.want {
			t.Errorf("Grade(%d).String() = %q, want %q", int(tt.grade), tt.grade.String(), tt.want)
		}
	}
}

func TestParseGradeRoundtrip(t *testing.T) {
	grades := []Grade{GradeForgot, GradeHard, GradeGood, GradeEasy}
	for _, g := range grades {
		parsed, err := ParseGrade(g.String())
		if err != nil {
			t.Errorf("ParseGrade(%q): %v", g.String(), err)
			continue
		}
		if parsed != g {
			t.Errorf("ParseGrade(%q) = %v, want %v", g.String(), parsed, g)
		}
	}
}

func TestParseGradeJSON(t *testing.T) {
	tests := []struct {
		s    string
		want Grade
	}{
		{"Forgot", GradeForgot},
		{"Hard", GradeHard},
		{"Good", GradeGood},
		{"Easy", GradeEasy},
	}
	for _, tt := range tests {
		got, err := ParseGradeJSON(tt.s)
		if err != nil {
			t.Errorf("ParseGradeJSON(%q): %v", tt.s, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseGradeJSON(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestInvalidGradeString(t *testing.T) {
	invalids := []string{"", "invalid"}
	for _, s := range invalids {
		if _, err := ParseGrade(s); err == nil {
			t.Errorf("ParseGrade(%q): expected error, got nil", s)
		}
	}
}

// TestUpdatePerformanceNewCard matches Rust's test_update_new_card.
func TestUpdatePerformanceNewCard(t *testing.T) {
	reviewedAt := types.Now()
	perf := types.NewCardPerformance()
	result := UpdatePerformance(perf, GradeGood, reviewedAt, DefaultFSRSConfig)

	if !result.LastReviewedAt.Equal(reviewedAt) {
		t.Errorf("LastReviewedAt = %v, want %v", result.LastReviewedAt, reviewedAt)
	}
	if !approxEq(result.Stability, 3.17) {
		t.Errorf("Stability = %f, want ≈ 3.17", result.Stability)
	}
	if !approxEq(result.Difficulty, 5.28) {
		t.Errorf("Difficulty = %f, want ≈ 5.28", result.Difficulty)
	}
	if !approxEq(result.IntervalRaw, 3.17) {
		t.Errorf("IntervalRaw = %f, want ≈ 3.17", result.IntervalRaw)
	}
	if result.IntervalDays != 3 {
		t.Errorf("IntervalDays = %d, want 3", result.IntervalDays)
	}
	if result.ReviewCount != 1 {
		t.Errorf("ReviewCount = %d, want 1", result.ReviewCount)
	}
}

// TestUpdatePerformanceAlreadyReviewed matches Rust's test_update_already_reviewed_card.
func TestUpdatePerformanceAlreadyReviewed(t *testing.T) {
	now := types.Now()
	lastReviewedAt := types.NewTimestamp(now.Time().Add(-3 * 24 * time.Hour))
	dueDate := types.NewDate(lastReviewedAt.Time().Add(3 * 24 * time.Hour))

	rp := types.ReviewedPerformance{
		LastReviewedAt: lastReviewedAt,
		Stability:      3.17,
		Difficulty:     5.28,
		IntervalRaw:    3.17,
		IntervalDays:   3,
		DueDate:        dueDate,
		ReviewCount:    1,
	}
	perf := types.ReviewedCardPerformance(rp)
	result := UpdatePerformance(perf, GradeEasy, now, DefaultFSRSConfig)

	if !result.LastReviewedAt.Equal(now) {
		t.Errorf("LastReviewedAt = %v, want %v", result.LastReviewedAt, now)
	}
	if !approxEq(result.Stability, 25.80) {
		t.Errorf("Stability = %f, want ≈ 25.80", result.Stability)
	}
	if !approxEq(result.Difficulty, 4.50) {
		t.Errorf("Difficulty = %f, want ≈ 4.50", result.Difficulty)
	}
	if !approxEq(result.IntervalRaw, 25.80) {
		t.Errorf("IntervalRaw = %f, want ≈ 25.80", result.IntervalRaw)
	}
	if result.IntervalDays != 26 {
		t.Errorf("IntervalDays = %d, want 26", result.IntervalDays)
	}
	if result.ReviewCount != 2 {
		t.Errorf("ReviewCount = %d, want 2", result.ReviewCount)
	}
}
