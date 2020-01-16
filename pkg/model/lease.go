package model

import (
	"fmt"
	"strings"
)

// Lease is a type corresponding to a Lease
// table record
type Lease struct {
	AccountID                *string                 `json:"accountId,omitempty" dynamodbav:"AccountId"`                                         // AWS Account ID
	PrincipalID              *string                 `json:"principalId,omitempty" dynamodbav:"PrincipalId"`                                     // Azure User Principal ID
	ID                       *string                 `json:"id,omitempty" dynamodbav:"Id,omitempty"`                                             // Lease ID
	LeaseStatus              *LeaseStatus            `json:"leaseStatus,omitempty" dynamodbav:"LeaseStatus,omitempty"`                           // Status of the Lease
	LeaseStatusReason        *LeaseStatusReason      `json:"leaseStatusReason,omitempty" dynamodbav:"LeaseStatusReason,omitempty"`               // Reason for the status of the lease
	CreatedOn                *int64                  `json:"createdOn,omitempty" dynamodbav:"CreatedOn,omitempty"`                               // Created Epoch Timestamp
	LastModifiedOn           *int64                  `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn,omitempty"`                     // Last Modified Epoch Timestamp
	BudgetAmount             *float64                `json:"budgetAmount,omitempty" dynamodbav:"BudgetAmount,omitempty"`                         // Budget Amount allocated for this lease
	BudgetCurrency           *string                 `json:"budgetCurrency,omitempty" dynamodbav:"BudgetCurrency,omitempty"`                     // Budget currency
	BudgetNotificationEmails *[]string               `json:"budgetNotificationEmails,omitempty" dynamodbav:"BudgetNotificationEmails,omitempty"` // Budget notification emails
	LeaseStatusModifiedOn    *int64                  `json:"leaseStatusModifiedOn,omitempty" dynamodbav:"LeaseStatusModifiedOn,omitempty"`       // Last Modified Epoch Timestamp
	ExpiresOn                *int64                  `json:"expiresOn,omitempty" dynamodbav:"ExpiresOn,omitempty"`                               // Lease expiration time as Epoch
	Metadata                 *map[string]interface{} `json:"metadata,omitempty" dynamodbav:"Metadata,omitempty"`                                 // Arbitrary key-value metadata to store with lease object
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
