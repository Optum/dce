package errors

import "errors"

// ErrBadRequest for poorly formatted requests
var ErrBadRequest = errors.New("bad request")

// ErrValidation for errors doing validation of the data
var ErrValidation = errors.New("validation error")

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
