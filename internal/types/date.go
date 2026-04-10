package types

import (
	"encoding/json"
	"time"

	"github.com/asano69/hashcards/internal/errs"
)

// dateLayout is the canonical date format used for storage and display.
const dateLayout = "2006-01-02"

// Date represents a calendar date without timezone or time-of-day information.
type Date struct {
	t time.Time
}

// NewDate creates a Date from a time.Time, discarding any sub-day precision.
func NewDate(t time.Time) Date {
	return Date{t: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

// Today returns the current date in UTC.
func Today() Date {
	now := time.Now().UTC()
	return Date{t: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)}
}

// ParseDate parses a "YYYY-MM-DD" formatted string.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, errs.Newf("invalid date: %s", s)
	}
	return Date{t: t}, nil
}

// Time returns the underlying time.Time value.
func (d Date) Time() time.Time {
	return d.t
}

// String returns the date formatted as "YYYY-MM-DD".
func (d Date) String() string {
	return d.t.Format(dateLayout)
}

// Equal returns true if d and other are the same calendar date.
func (d Date) Equal(other Date) bool {
	return d.t.Equal(other.t)
}

// Before returns true if d is strictly earlier than other.
func (d Date) Before(other Date) bool {
	return d.t.Before(other.t)
}

// LessOrEqual returns true if d is on or before other.
func (d Date) LessOrEqual(other Date) bool {
	return !d.t.After(other.t)
}

// MarshalJSON serializes Date as a "YYYY-MM-DD" JSON string.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON deserializes a Date from a "YYYY-MM-DD" JSON string.
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
