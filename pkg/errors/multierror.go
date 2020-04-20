package errors

import (
	"fmt"
	"io"
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
	for _, err := range e.Errors {
		errStrs = append(errStrs, err.Error())
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

// Format for the standard format library
func (e MultiError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%s\n", e.Message)
			for _, err := range e.Errors {
				_, _ = fmt.Fprintf(s, "%+v", err)
			}
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}
