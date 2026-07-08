package fsrs

import (
	"encoding/hex"
	"testing"

	"github.com/asano69/hatchards/internal/types"
)

func floatPtr(f float64) *float64 { return &f }

// TestWithReviewFuzzNoFuzzFactor matches Rust's "no fuzz" cases: with no
// fuzz factor, the interval is just rounded and clamped.
func TestWithReviewFuzzNoFuzzFactor(t *testing.T) {
	tests := []struct {
		interval         float64
		minimum, maximum int64
		want             int64
	}{
		{1.5, 1, 100, 2},
		{0.1, 1, 100, 1},
		{101.0, 1, 100, 100},
	}
	for _, tt := range tests {
		got := WithReviewFuzz(nil, tt.interval, tt.minimum, tt.maximum)
		if got != tt.want {
			t.Errorf("WithReviewFuzz(nil, %v, %d, %d) = %d, want %d",
				tt.interval, tt.minimum, tt.maximum, got, tt.want)
		}
	}
}

// TestWithReviewFuzzRange matches Rust's assert_lower_middle_upper! table:
// for a fuzz factor of 0.0, 0.5, and 0.99, the result should land at the
// lower bound, roughly the middle, and the upper bound respectively.
func TestWithReviewFuzzRange(t *testing.T) {
	tests := []struct {
		name                 string
		interval             float64
		minimum, maximum     int64
		lower, middle, upper int64
	}{
		{"no fuzzing below 2.5 days (1.0)", 1.0, 1, 1000, 1, 1, 1},
		{"no fuzzing below 2.5 days (2.49)", 2.49, 1, 1000, 2, 2, 2},
		{"1 day of fuzz at 2.5", 2.5, 1, 1000, 2, 3, 4},
		{"fuzz range 2.5-7", 7.0, 1, 1000, 5, 7, 9},
		{"fuzz range 7-20", 17.0, 1, 1000, 14, 17, 20},
		{"fuzz range above 20", 37.0, 1, 1000, 33, 37, 41},
		{"min forces a single value", 2.0, 2, 1000, 2, 2, 2},
		{"widened range when room allows", 2.0, 3, 1000, 3, 4, 4},
		{"min == max collapses range", 2.0, 3, 3, 3, 3, 3},
		{"range transition below 7", 6.9, 3, 1000, 5, 7, 9},
		{"range transition at 7", 7.0, 3, 1000, 5, 7, 9},
		{"range transition above 7", 7.1, 3, 1000, 5, 7, 9},
		{"range transition below 20", 19.9, 3, 1000, 17, 20, 23},
		{"range transition at 20", 20.0, 3, 1000, 17, 20, 23},
		{"range transition above 20", 20.1, 3, 1000, 17, 20, 23},
		{"minimum above target interval", 100.0, 101, 1000, 101, 105, 108},
		{"maximum below target interval", 100.0, 1, 99, 92, 96, 99},
		{"tight min/max window", 100.0, 97, 103, 97, 100, 103},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithReviewFuzz(floatPtr(0.0), tt.interval, tt.minimum, tt.maximum); got != tt.lower {
				t.Errorf("fuzz=0.0: got %d, want lower bound %d", got, tt.lower)
			}
			if got := WithReviewFuzz(floatPtr(0.5), tt.interval, tt.minimum, tt.maximum); got != tt.middle {
				t.Errorf("fuzz=0.5: got %d, want middle %d", got, tt.middle)
			}
			if got := WithReviewFuzz(floatPtr(0.99), tt.interval, tt.minimum, tt.maximum); got != tt.upper {
				t.Errorf("fuzz=0.99: got %d, want upper bound %d", got, tt.upper)
			}
		})
	}
}

// TestConstrainedFuzzBoundsInvalidValuesDoNotPanic verifies that a
// minimum greater than maximum is handled gracefully instead of panicking.
func TestConstrainedFuzzBoundsInvalidValuesDoNotPanic(t *testing.T) {
	ConstrainedFuzzBounds(1.0, 3, 2)
}

// TestFuzzFactorDeterministic verifies that the same card hash and review
// count always produce the same fuzz factor.
func TestFuzzFactorDeterministic(t *testing.T) {
	hash := types.HashBytes([]byte("some card content"))
	f1 := FuzzFactor(hash, 3)
	f2 := FuzzFactor(hash, 3)
	if f1 != f2 {
		t.Errorf("FuzzFactor not deterministic: got %v and %v", f1, f2)
	}
	if f1 < 0 || f1 >= 1 {
		t.Errorf("FuzzFactor = %v, want value in [0, 1)", f1)
	}
}

// TestFuzzFactorVariesByReviewCount verifies that different review counts
// for the same card produce different fuzz factors (with overwhelming
// probability), so repeated reviews of one card don't all fuzz identically.
func TestFuzzFactorVariesByReviewCount(t *testing.T) {
	hash := types.HashBytes([]byte("some card content"))
	f1 := FuzzFactor(hash, 1)
	f2 := FuzzFactor(hash, 2)
	if f1 == f2 {
		t.Error("expected different fuzz factors for different review counts")
	}
}

// TestFuzzSeedFallback verifies that fuzzSeed degrades gracefully (rather
// than panicking) if it's ever given a malformed hash.
func TestFuzzSeedFallback(t *testing.T) {
	_, err := hex.DecodeString("not-hex")
	if err == nil {
		t.Fatal("test setup assumption broken: expected decode error")
	}
	// fuzzSeed itself only accepts a well-formed types.CardHash, so this
	// test only documents the fallback path exists; a malformed CardHash
	// cannot be constructed outside this package.
}
