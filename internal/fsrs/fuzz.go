// fuzz.go adds a small random variation to review intervals, so cards
// scheduled around the same time don't all pile up on the same day. The
// algorithm mirrors Anki's implementation in
// rslib/src/scheduler/states/fuzz.rs.
package fsrs

import (
	"encoding/binary"
	"encoding/hex"
	"math"

	"github.com/asano69/hatchards/internal/rng"
	"github.com/asano69/hatchards/internal/types"
)

// fuzzRange describes a range of interval lengths (in days) and the
// fraction of the interval that is added to the fuzz delta within it.
type fuzzRange struct {
	start, end, factor float64
}

// fuzzRanges mirrors Anki's FUZZ_RANGES: fuzz grows with the interval
// length, but at a decreasing rate for longer intervals.
var fuzzRanges = []fuzzRange{
	{start: 2.5, end: 7.0, factor: 0.15},
	{start: 7.0, end: 20.0, factor: 0.1},
	{start: 20.0, end: math.MaxFloat64, factor: 0.05},
}

// fuzzDelta returns the amount of fuzz to apply to interval, in both
// directions. Intervals shorter than 2.5 days are never fuzzed.
func fuzzDelta(interval float64) float64 {
	if interval < 2.5 {
		return 0
	}
	delta := 1.0
	for _, r := range fuzzRanges {
		clamped := math.Min(interval, r.end)
		if span := clamped - r.start; span > 0 {
			delta += r.factor * span
		}
	}
	return delta
}

// fuzzBounds returns the [lower, upper] interval (in days) that fuzz may
// produce for the given target interval, before clamping to minimum/maximum.
func fuzzBounds(interval float64) (lower, upper int64) {
	delta := fuzzDelta(interval)
	return int64(math.Round(interval - delta)), int64(math.Round(interval + delta))
}

func clampInt64(v, min, max int64) int64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ConstrainedFuzzBounds returns the bounds of the fuzz range, clamped to
// [minimum, maximum]. If the range collapses to a single day but there's
// still room to widen it, the upper bound is nudged up by one, so a fuzz
// factor of 0 and a fuzz factor near 1 don't produce the same interval.
func ConstrainedFuzzBounds(interval float64, minimum, maximum int64) (lower, upper int64) {
	if minimum > maximum {
		minimum = maximum
	}
	interval = math.Max(float64(minimum), math.Min(float64(maximum), interval))

	lower, upper = fuzzBounds(interval)
	lower = clampInt64(lower, minimum, maximum)
	upper = clampInt64(upper, minimum, maximum)
	if upper == lower && upper > 2 && upper < maximum {
		upper = lower + 1
	}
	return lower, upper
}

// WithReviewFuzz applies fuzzFactor (expected to be in [0, 1)) to interval,
// respecting the minimum/maximum bounds. A nil fuzzFactor disables fuzzing
// entirely, falling back to a plain rounded-and-clamped interval.
func WithReviewFuzz(fuzzFactor *float64, interval float64, minimum, maximum int64) int64 {
	if fuzzFactor == nil {
		return clampInt64(int64(math.Round(interval)), minimum, maximum)
	}
	lower, upper := ConstrainedFuzzBounds(interval, minimum, maximum)
	return int64(math.Floor(float64(lower) + *fuzzFactor*float64(1+upper-lower)))
}

// FuzzFactor derives a deterministic fuzz factor in [0, 1) from a card's
// hash and its review count, so the same card+review always fuzzes the
// same way (e.g. across a page reload), while different cards, and
// different reviews of the same card, fuzz independently.
func FuzzFactor(cardHash types.CardHash, reviewCount int) float64 {
	r := rng.FromSeed(fuzzSeed(cardHash, reviewCount))
	return float64(r.NextU32()) / (float64(math.MaxUint32) + 1)
}

// fuzzSeed folds a card hash and review count into a single uint64 seed.
func fuzzSeed(cardHash types.CardHash, reviewCount int) uint64 {
	raw, err := hex.DecodeString(cardHash.Hex())
	if err != nil || len(raw) < 8 {
		return uint64(reviewCount)
	}
	return binary.BigEndian.Uint64(raw[:8]) + uint64(reviewCount)
}
