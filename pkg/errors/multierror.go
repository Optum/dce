package errors

import (
	"fmt"
	"strings"
)

// MultiError is an Error type that wraps multiple errors.
// This can be a useful way to combine errors in a method
// where you want to allow process to continue through multiple
// failed steps.
type MultiError struct {
	Message string
	Errors  []error
}

// Error returns the error message to satisfy the error interface
func (e MultiError) Error() string {
	var errStrs []string
	for _, e := range e.Errors {
		errStrs = append(errStrs, e.Error())
	}
	return fmt.Sprintf(
		"%s: %s",
		e.Message,
		strings.Join(errStrs, "; "),
	)
}

// Is to satisfy the error comparison interface
func (e MultiError) Is(err error) bool {
	return e.Error() == err.Error()
}

// NewMultiError is a list of errors
func NewMultiError(msg string, errs []error) *MultiError {
	return &MultiError{
		Message: msg,
		Errors:  errs,
	}
}
