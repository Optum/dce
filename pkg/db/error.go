package db

// StatusTransitionError means that we failed to transition
// an Account or Lease from one status to another,
// likely because the prevStatus condition was not met
type StatusTransitionError struct {
	err string
}

func (e *StatusTransitionError) Error() string {
	return e.err
}

// AccountLeasedError is returned when a consumer attempts to delete an account that is currently at status Leased
type AccountLeasedError struct {
	err string
}

func (e *AccountLeasedError) Error() string {
	return e.err
}

// AccountNotFoundError is returned when an account is not found.
type AccountNotFoundError struct {
	err string
}

func (e *AccountNotFoundError) Error() string {
	return e.err
}


// NotFoundError is returned when a resource is not found.
type NotFoundError struct {
	Err string
}

func (e *NotFoundError) Error() string {
	return e.Err
}
