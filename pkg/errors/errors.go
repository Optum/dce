package errors

import "errors"

// ErrBadRequest for poorly formatted requests
var ErrBadRequest = errors.New("bad request")

// ErrValidation for errors doing validation of the data
type ErrValidation struct {
	Message string
	Err     error
}

func (e *ErrValidation) Error() string { return e.Message }

// Unwrap provides the underlying error
func (e *ErrValidation) Unwrap() error { return e.Err }

// Cause provides the underlying error
func (e *ErrValidation) Cause() error { return e.Err }

// Is helps with new errors validation
func (e *ErrValidation) Is(target error) bool {
	t, ok := target.(*ErrValidation)
	if !ok {
		return false
	}
	return (e.Message == t.Message || t.Message == "")
}

// ErrNotFound resource not found
var ErrNotFound = errors.New("not found")

// ErrInternalServer for errors doing internal processing
var ErrInternalServer = errors.New("internal server error")

// ErrConflict is that there was a conflict in processing the request
var ErrConflict = errors.New("conflict found")

// ErrAccountIsLeased is when the action can't be taken because the account is in use
var ErrAccountIsLeased = errors.New("the account is currently leased")

// ErrAccountStatusChange for when the Account can't change its status
var ErrAccountStatusChange = errors.New("the account status could not be transitioned")
