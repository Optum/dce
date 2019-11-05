package db

import (
	"fmt"
	"strings"
)

// RedboxAccount is a type corresponding to a RedboxAccount table record
type RedboxAccount struct {
	ID                  string                 `json:"Id"`             // AWS Account ID
	AccountStatus       AccountStatus          `json:"AccountStatus"`  // Status of the AWS Account
	LastModifiedOn      int64                  `json:"LastModifiedOn"` // Last Modified Epoch Timestamp
	CreatedOn           int64                  `json:"CreatedOn"`
	AdminRoleArn        string                 `json:"AdminRoleArn"`        // Assumed by the Redbox master account, to manage this user account
	PrincipalRoleArn    string                 `json:"PrincipalRoleArn"`    // Assumed by principal users of Redbox
	PrincipalPolicyHash string                 `json:"PrincipalPolicyHash"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"Metadata"`            // Any org specific metadata pertaining to the account
}

// RedboxLease is a type corresponding to a RedboxLease
// table record
type RedboxLease struct {
	AccountID                string            `json:"AccountId"`                // AWS Account ID
	PrincipalID              string            `json:"PrincipalId"`              // Azure User Principal ID
	ID                       string            `json:"Id"`                       // Lease ID
	LeaseStatus              LeaseStatus       `json:"LeaseStatus"`              // Status of the Lease
	LeaseStatusReason        LeaseStatusReason `json:"LeaseStatusReason"`        // Reason for the status of the lease
	CreatedOn                int64             `json:"CreatedOn"`                // Created Epoch Timestamp
	LastModifiedOn           int64             `json:"LastModifiedOn"`           // Last Modified Epoch Timestamp
	BudgetAmount             float64           `json:"BudgetAmount"`             // Budget Amount allocated for this lease
	BudgetCurrency           string            `json:"BudgetCurrency"`           // Budget currency
	BudgetNotificationEmails []string          `json:"BudgetNotificationEmails"` // Budget notification emails
	LeaseStatusModifiedOn    int64             `json:"LeaseStatusModifiedOn"`    // Last Modified Epoch Timestamp
	ExpiresOn                int64             `json:"ExpiresOn"`                // Lease expiration time as Epoch
}

// Timestamp is a timestamp type for epoch format
type Timestamp int64

// Timestamped contains timestamp types
type Timestamped struct {
	CreatedOn      Timestamp
	LastModifiedOn Timestamp
}

// AccountStatus is a Redbox account status type
type AccountStatus string

const (
	// None status
	None AccountStatus = "None"
	// Ready status
	Ready AccountStatus = "Ready"
	// NotReady status
	NotReady AccountStatus = "NotReady"
	// Leased status
	Leased AccountStatus = "Leased"
)

// ParseAccountStatus - parses the string into an account status.
func ParseAccountStatus(status string) (AccountStatus, error) {
	switch strings.ToLower(status) {
	case "ready":
		return Ready, nil
	case "notready":
		return NotReady, nil
	case "leased":
		return Leased, nil
	}
	return None, fmt.Errorf("Cannot parse value %s", status)
}

// LeaseStatus is a Redbox account lease status type
type LeaseStatus string

const (
	// Active status
	Active LeaseStatus = "Active"
	// Inactive status
	Inactive LeaseStatus = "Inactive"
)

// LeaseStatusReason provides consistent verbiage for lease status change reasons.
type LeaseStatusReason string

const (
	// LeaseExpired means the lease has past its expiresOn date and therefore expired.
	LeaseExpired LeaseStatusReason = "Expired"
	// LeaseOverBudget means the lease is over its budgeted amount and is therefore reset/reclaimed.
	LeaseOverBudget LeaseStatusReason = "OverBudget"
	// LeaseDestroyed means the lease has been deleted via an API call or other user action.
	LeaseDestroyed LeaseStatusReason = "Destroyed"
	// LeaseActive means the lease is still active.
	LeaseActive LeaseStatusReason = "Active"
	// LeaseRolledBack means something happened in the system that caused the lease to be inactive
	// based on an error happening and rollback occuring
	LeaseRolledBack LeaseStatusReason = "Rollback"
)
