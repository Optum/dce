package errors

import "errors"

var ErrValidation = errors.New("validation error")
var ErrNotFound = errors.New("not found")
var ErrInternalServer = errors.New("internal server error")
var ErrConflict = errors.New("conflict found")

var ErrAccountIsLeased = errors.New("the account is currently leased")
var ErrAccountStatusChange = errors.New("the account status could not be transitioned")
