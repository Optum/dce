package dceerr

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

const (
	clientError        = "ClientError"
	serverError        = "ServerError"
	validationError    = "RequestValidationError"
	alreadyExistsError = "AlreadyExistsError"
	notFoundError      = "NotFoundError"
	conflictError      = "ConflictError"
)

type detailError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// StatusError is the custom error type we are using.
// Should satisfy errors interface
type StatusError struct {
	apiCode int
	cause   error
	Details detailError `json:"error"`
	stack   *stack
}

func (e *StatusError) Error() string { return e.Details.Message }

// OriginalError provides the underlying error
func (e *StatusError) OriginalError() error { return e.cause }

// APICode returns the http code
func (e *StatusError) APICode() int { return e.apiCode }

// StackTrace returns the frames for a stack trace
func (e *StatusError) StackTrace() errors.StackTrace {
	return e.stack.StackTrace()
}

// Format for the standard format library
func (e *StatusError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v", e.OriginalError())
			e.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// APICode returns the API Code
type APICode interface {
	APICode() int
}

// APICodeForError returns the HTTP status for a particular error.
func APICodeForError(err error) int {
	switch t := err.(type) {
	case APICode:
		return t.APICode()
	}
	return http.StatusInternalServerError
}

// GetStackTrace returns the API Code
type GetStackTrace interface {
	StackTrace() errors.StackTrace
}

// GetStackTraceForError returns the HTTP status for a particular error.
func GetStackTraceForError(err error) errors.StackTrace {
	switch t := err.(type) {
	case GetStackTrace:
		return t.StackTrace()
	}
	return nil
}

// NewValidation creates a validation error
func NewValidation(group string, err error) *StatusError {
	return &StatusError{
		apiCode: http.StatusBadRequest,
		cause:   err,
		Details: detailError{
			Message: fmt.Sprintf("%s validation error: %v", group, err),
			Code:    validationError,
		},
		stack: callers(),
	}
}

// NewNotFound returns an a NotFound error with standard messaging
func NewNotFound(group string, name string) *StatusError {
	return &StatusError{
		apiCode: http.StatusNotFound,
		Details: detailError{
			Message: fmt.Sprintf("%s %q not found", group, name),
			Code:    notFoundError,
		},
		stack: callers(),
	}
}

// NewInternalServer returns an error for Internal Server Errors
func NewInternalServer(m string, err error) *StatusError {
	return &StatusError{
		apiCode: http.StatusInternalServerError,
		cause:   err,
		Details: detailError{
			Message: m,
			Code:    serverError,
		},
		stack: callers(),
	}
}

// NewConflict returns a new error for representing Conflicts
func NewConflict(group string, name string, err error) *StatusError {
	return &StatusError{
		apiCode: http.StatusConflict,
		cause:   err,
		Details: detailError{
			Message: fmt.Sprintf("operation cannot be fulfilled on %s %q: %v", group, name, err),
			Code:    conflictError,
		},
		stack: callers(),
	}
}

// NewBadRequest returns a new error representing a bad request
func NewBadRequest(m string) *StatusError {
	return &StatusError{
		apiCode: http.StatusBadRequest,
		cause:   nil,
		Details: detailError{
			Message: m,
			Code:    clientError,
		},
		stack: callers(),
	}
}

// NewServiceUnavailable returns a new error representing service unavailable
func NewServiceUnavailable(m string) *StatusError {
	return &StatusError{
		apiCode: http.StatusServiceUnavailable,
		cause:   nil,
		Details: detailError{
			Message: m,
			Code:    serverError,
		},
		stack: callers(),
	}
}

// NewAlreadyExists returns a new error representing an already exists error
func NewAlreadyExists(group string, name string) *StatusError {
	return &StatusError{
		apiCode: http.StatusConflict,
		cause:   nil,
		Details: detailError{
			Message: fmt.Sprintf("%s %q already exists", group, name),
			Code:    alreadyExistsError,
		},
		stack: callers(),
	}
}

// NewAdminRoleNotAssumable returns a new error representing an admin role not being assumable
func NewAdminRoleNotAssumable(role string, err error) *StatusError {
	return &StatusError{
		apiCode: http.StatusUnprocessableEntity,
		cause:   err,
		Details: detailError{
			Message: fmt.Sprintf("adminRole %q is not assumable by the parent account", role),
			Code:    validationError,
		},
		stack: callers(),
	}
}

// Cause gets the original error
func Cause(err error) error {
	type unwraper interface {
		Unwrap() error
	}

	for err != nil {
		cause, ok := err.(unwraper)
		if !ok {
			break
		}
		err = cause.Unwrap()
	}
	return err
}
