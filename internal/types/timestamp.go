package types

import (
	"encoding/json"
	"time"

	"github.com/asano69/hashcards/internal/errs"
)

// timestampLayout is the canonical timestamp format: millisecond precision, no timezone.
const timestampLayout = "2006-01-02T15:04:05.000"

// Timestamp is a point in time with millisecond precision, always stored in UTC.
type Timestamp struct {
	t time.Time
}

// NewTimestamp creates a Timestamp from a time.Time, converting to UTC and
// truncating to milliseconds.
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{t: t.UTC().Truncate(time.Millisecond)}
}

// Now returns the current time as a UTC Timestamp.
func Now() Timestamp {
	return NewTimestamp(time.Now())
}

// ParseTimestamp parses a "YYYY-MM-DDTHH:MM:SS.mmm" formatted string as UTC.
func ParseTimestamp(s string) (Timestamp, error) {
	t, err := time.Parse(timestampLayout, s)
	if err != nil {
		return Timestamp{}, errs.Newf("Failed to parse timestamp: '%s'.", s)
	}
	return Timestamp{t: t}, nil
}

// Time returns the underlying time.Time value (always UTC).
func (ts Timestamp) Time() time.Time {
	return ts.t
}

// String returns the timestamp in "YYYY-MM-DDTHH:MM:SS.mmm" format.
func (ts Timestamp) String() string {
	return ts.t.Format(timestampLayout)
}

// Date returns the date component of the timestamp.
func (ts Timestamp) Date() Date {
	return NewDate(ts.t)
}

// Equal returns true if ts and other represent the same point in time.
func (ts Timestamp) Equal(other Timestamp) bool {
	return ts.t.Equal(other.t)
}

// MarshalJSON serializes Timestamp as a JSON string.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(ts.String())
}

// UnmarshalJSON deserializes a Timestamp from a JSON string.
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseTimestamp(s)
	if err != nil {
		return err
	}
	*ts = parsed
	return nil
}
