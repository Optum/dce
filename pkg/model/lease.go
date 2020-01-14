package model

import (
	"fmt"
	"strings"
)

// Lease is a type corresponding to a Lease
// table record
type Lease struct {
	AccountID                *string                 `json:"AccountId"`                // AWS Account ID
	PrincipalID              *string                 `json:"PrincipalId"`              // Azure User Principal ID
	ID                       *string                 `json:"Id"`                       // Lease ID
	LeaseStatus              *LeaseStatus            `json:"LeaseStatus"`              // Status of the Lease
	LeaseStatusReason        *LeaseStatusReason      `json:"LeaseStatusReason"`        // Reason for the status of the lease
	CreatedOn                *int64                  `json:"CreatedOn"`                // Created Epoch Timestamp
	LastModifiedOn           *int64                  `json:"LastModifiedOn"`           // Last Modified Epoch Timestamp
	BudgetAmount             *float64                `json:"BudgetAmount"`             // Budget Amount allocated for this lease
	BudgetCurrency           *string                 `json:"BudgetCurrency"`           // Budget currency
	BudgetNotificationEmails *[]string               `json:"BudgetNotificationEmails"` // Budget notification emails
	LeaseStatusModifiedOn    *int64                  `json:"LeaseStatusModifiedOn"`    // Last Modified Epoch Timestamp
	ExpiresOn                *int64                  `json:"ExpiresOn"`                // Lease expiration time as Epoch
	Metadata                 *map[string]interface{} `json:"Metadata"`                 // Arbitrary key-value metadata to store with lease object
}

// Leases is multiple of Lease
type Leases []Lease

// LeaseStatus is a lease status type
type LeaseStatus string

const (
	// EmptyLeaseStatus status
	EmptyLeaseStatus LeaseStatus = ""
	// Active status
	LeaseStatusActive LeaseStatus = "Active"
	// Inactive status
	LeaseStatusInactive LeaseStatus = "Inactive"
)

// String returns the string value of LeaseStatus
func (c LeaseStatus) String() string {
	return string(c)
}

// StringPtr returns a pointer to the string value of LeaseStatus
func (c LeaseStatus) StringPtr() *string {
	v := string(c)
	return &v
}

// LeaseStatusPtr returns a pointer to the string value of LeaseStatus
func (c LeaseStatus) LeaseStatusPtr() *LeaseStatus {
	v := c
	return &v
}

// ParseLeaseStatus - parses the string into an account status.
func ParseLeaseStatus(status string) (LeaseStatus, error) {
	switch strings.ToLower(status) {
	case "active":
		return LeaseStatusActive, nil
	case "inactive":
		return LeaseStatusInactive, nil
	}
	return EmptyLeaseStatus, fmt.Errorf("Cannot parse value %s", status)
}

// LeaseStatusReason provides consistent verbiage for lease status change reasons.
type LeaseStatusReason string

const (
	// LeaseExpired means the lease has past its expiresOn date and therefore expired.
	LeaseExpired LeaseStatusReason = "Expired"
	// LeaseOverBudget means the lease is over its budgeted amount and is therefore reset/reclaimed.
	LeaseOverBudget LeaseStatusReason = "OverBudget"
	// LeaseOverPrincipalBudget means the lease is over its principal budgeted amount and is therefore reset/reclaimed.
	LeaseOverPrincipalBudget LeaseStatusReason = "OverPrincipalBudget"
	// LeaseDestroyed means the lease has been deleted via an API call or other user action.
	LeaseDestroyed LeaseStatusReason = "Destroyed"
	// LeaseActive means the lease is still active.
	LeaseActive LeaseStatusReason = "Active"
	// LeaseRolledBack means something happened in the system that caused the lease to be inactive
	// based on an error happening and rollback occuring
	LeaseRolledBack LeaseStatusReason = "Rollback"
	// AccountOrphaned means that the health of the account was compromised.  The account has been orphaned
	// which means the leases are also made Inactive
	AccountOrphaned LeaseStatusReason = "AccountOrphaned"
)
