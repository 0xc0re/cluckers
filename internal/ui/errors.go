package ui

import (
	"errors"
	"strings"
)

// UserError is a user-facing error with optional technical detail and actionable suggestion.
type UserError struct {
	Message    string // User-friendly message (always shown).
	Detail     string // Technical detail (shown with -v).
	Suggestion string // Actionable hint (shown if non-empty).
	Err        error  // Wrapped underlying error.
}

// Error implements the error interface, returning the user-friendly message.
func (e *UserError) Error() string {
	return e.Message
}

// Unwrap returns the wrapped error for errors.Is/As compatibility.
func (e *UserError) Unwrap() error {
	return e.Err
}

// FormatError formats an error for display. If it is a UserError, the detail
// and suggestion are included based on verbose mode. Plain errors show their
// string representation.
func FormatError(err error, verbose bool) string {
	var ue *UserError
	if !errors.As(err, &ue) {
		return err.Error()
	}

	var b strings.Builder
	b.WriteString(ue.Message)

	if verbose && ue.Detail != "" {
		b.WriteString("\n  Detail: ")
		b.WriteString(ue.Detail)
	}
	if ue.Suggestion != "" {
		b.WriteString("\n  Suggestion: ")
		b.WriteString(ue.Suggestion)
	}
	return b.String()
}
