package db

// StatusTransitionError means that we failed to transition
// an Account or Assignment from one status to another,
// likely because the prevStatus condition was not met
type StatusTransitionError struct {
	err string
}

func (e *StatusTransitionError) Error() string {
	return e.err
}

// AccountAssignedError is returned when a consumer attempts to delete an account that is currently at status Assigned
type AccountAssignedError struct {
	err string
}

func (e *AccountAssignedError) Error() string {
	return e.err
}

// AccountNotFoundError is returned when an account is not found.
type AccountNotFoundError struct {
	err string
}

func (e *AccountNotFoundError) Error() string {
	return e.err
}
