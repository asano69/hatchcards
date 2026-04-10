package types

// Performance represents the FSRS learning state of a card.
// A card is either new (never reviewed) or reviewed (has scheduling data).
type Performance struct {
	isNew    bool
	reviewed *ReviewedPerformance
}

// ReviewedPerformance holds the FSRS scheduling parameters for a reviewed card.
type ReviewedPerformance struct {
	LastReviewedAt Timestamp
	Stability      float64
	Difficulty     float64
	IntervalRaw    float64
	IntervalDays   int64
	DueDate        Date
	ReviewCount    int
}

// NewCardPerformance returns a Performance for a card that has never been reviewed.
func NewCardPerformance() Performance {
	return Performance{isNew: true}
}

// ReviewedCardPerformance returns a Performance wrapping scheduling data.
func ReviewedCardPerformance(rp ReviewedPerformance) Performance {
	return Performance{isNew: false, reviewed: &rp}
}

// IsNew returns true if the card has never been reviewed.
func (p Performance) IsNew() bool {
	return p.isNew
}

// Reviewed returns the scheduling data, or nil if the card is new.
func (p Performance) Reviewed() *ReviewedPerformance {
	return p.reviewed
}
