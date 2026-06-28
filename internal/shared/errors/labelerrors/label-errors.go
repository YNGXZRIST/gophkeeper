// Package labelerrors provides labeled error wrappers.
package labelerrors

import "fmt"

// LabelError wraps an error with a logical subsystem label.
type LabelError struct {
	Label string
	Err   error
}

// Error returns string representation of labeled error.
func (e LabelError) Error() string {
	return fmt.Sprintf("[%s]: %s", e.Label, e.Err)
}

// NewLabelError creates labeled error wrapper.
func NewLabelError(label string, err error) error {
	return LabelError{Label: label, Err: err}
}

// Unwrap returns underlying wrapped error.
func (e LabelError) Unwrap() error { return e.Err }
