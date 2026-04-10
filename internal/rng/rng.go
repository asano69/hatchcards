// Package rng provides a minimal linear congruential PRNG used to shuffle cards.
// It is intentionally insecure and exists solely for reproducible, dependency-free
// shuffling, matching the Rust implementation exactly.
package rng

// LCG constants, identical to the Rust implementation.
const (
	lcgA uint64 = 6364136223846793005
	lcgC uint64 = 1442695040888963407
)

// TinyRng is a zero-dependency linear congruential pseudo-random number
// generator. It is not suitable for cryptographic use.
type TinyRng struct {
	state uint64
}

// FromSeed creates a new TinyRng initialised with the given seed.
func FromSeed(seed uint64) *TinyRng {
	return &TinyRng{state: seed}
}

// NextU32 advances the LCG state and returns the high 32 bits of the new state.
func (r *TinyRng) NextU32() uint32 {
	r.state = r.state*lcgA + lcgC
	return uint32(r.state >> 32)
}

// Generate returns a pseudo-random uint32 in the range [0, max).
func (r *TinyRng) Generate(max uint32) uint32 {
	return r.NextU32() % max
}

// Shuffle returns a new slice containing all elements of s in a pseudo-random
// order determined by rng. The original slice is not modified.
// The algorithm matches the Rust implementation: for each index i, swap
// element i with a randomly chosen index j in [0, len).
func Shuffle[T any](s []T, rng *TinyRng) []T {
	result := make([]T, len(s))
	copy(result, s)
	n := uint32(len(result))
	for i := uint32(0); i < n; i++ {
		j := rng.Generate(n)
		result[i], result[j] = result[j], result[i]
	}
	return result
}
