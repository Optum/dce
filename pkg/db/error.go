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
