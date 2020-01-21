package lease

import (
	"fmt"
	"strings"
)

// Status is a lease status type
type Status string

const (
	// EmptyStatus status
	EmptyStatus Status = ""
	// StatusActive status
	StatusActive Status = "Active"
	// StatusInactive status
	StatusInactive Status = "Inactive"
)

// String returns the string value of Status
func (c Status) String() string {
	return string(c)
}

// StringPtr returns a pointer to the string value of Status
func (c Status) StringPtr() *string {
	v := string(c)
	return &v
}

// StatusPtr returns a pointer to the string value of Status
func (c Status) StatusPtr() *Status {
	v := c
	return &v
}

// ParseStatus - parses the string into an account status.
func ParseStatus(status string) (Status, error) {
	switch strings.ToLower(status) {
	case "active":
		return StatusActive, nil
	case "inactive":
		return StatusInactive, nil
	}
	return EmptyStatus, fmt.Errorf("Cannot parse value %s", status)
}

// StatusReason provides consistent verbiage for lease status change reasons.
type StatusReason string

const (
	// StatusReasonExpired means the lease has past its expiresOn date and therefore expired.
	StatusReasonExpired StatusReason = "Expired"
	// StatusReasonOverBudget means the lease is over its budgeted amount and is therefore reset/reclaimed.
	StatusReasonOverBudget StatusReason = "OverBudget"
	// StatusReasonOverPrincipalBudget means the lease is over its principal budgeted amount and is therefore reset/reclaimed.
	StatusReasonOverPrincipalBudget StatusReason = "OverPrincipalBudget"
	// StatusReasonDestroyed means the lease has been deleted via an API call or other user action.
	StatusReasonDestroyed StatusReason = "Destroyed"
	// StatusReasonActive means the lease is still active.
	StatusReasonActive StatusReason = "Active"
	// StatusReasonRolledBack means something happened in the system that caused the lease to be inactive
	// based on an error happening and rollback occuring
	StatusReasonRolledBack StatusReason = "Rollback"
	// StatusReasonAccountOrphaned means that the health of the account was compromised.  The account has been orphaned
	// which means the leases are also made Inactive
	StatusReasonAccountOrphaned StatusReason = "LeaseAccountOrphaned"
)
