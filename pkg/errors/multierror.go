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

func (err MultiError) Error() string {
	var errStrs []string
	for _, e := range err.Errors {
		errStrs = append(errStrs, e.Error())
	}
	return fmt.Sprintf(
		"%s: %s",
		err.Message,
		strings.Join(errStrs, "; "),
	)
}

func NewMultiError(msg string, errs []error) *MultiError {
	return &MultiError{
		Message: msg,
		Errors:  errs,
	}
}
