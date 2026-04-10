// Package errs provides the standard error type for hashcards.
package errs

import "fmt"

// ErrorReport is the standard error type used throughout the application.
// It wraps a plain message string and implements the error interface.
type ErrorReport struct {
	message string
}

// New creates a new ErrorReport with the given message.
func New(msg string) *ErrorReport {
	return &ErrorReport{message: msg}
}

// Newf creates a new ErrorReport with a formatted message.
func Newf(format string, args ...any) *ErrorReport {
	return &ErrorReport{message: fmt.Sprintf(format, args...)}
}

// Error implements the error interface.
// The format matches the Rust implementation: "error: <message>".
func (e *ErrorReport) Error() string {
	return "error: " + e.message
}
