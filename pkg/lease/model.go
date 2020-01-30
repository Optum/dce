package lease

import (
	"fmt"
	"strings"

	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Lease is a type corresponding to a Lease
// table record
type Lease struct {
	AccountID                *string                `json:"accountId,omitempty" dynamodbav:"AccountId" schema:"accountId,omitempty"`                                         // AWS Account ID
	PrincipalID              *string                `json:"principalId,omitempty" dynamodbav:"PrincipalId" schema:"principalId,omitempty"`                                     // Azure User Principal ID
	ID                       *string                `json:"id,omitempty" dynamodbav:"Id,omitempty" schema:"id,omitempty"`                                             // Lease ID
	Status                   *Status                `json:"leaseStatus,omitempty" dynamodbav:"LeaseStatus,omitempty" schema:"status,omitempty"`                           // Status of the Lease
	StatusReason             *StatusReason          `json:"leaseStatusReason,omitempty" dynamodbav:"LeaseStatusReason,omitempty" schema:"leaseStatusReason,omitempty"`               // Reason for the status of the lease
	CreatedOn                *int64                 `json:"createdOn,omitempty" dynamodbav:"CreatedOn,omitempty" schema:"createdOn,omitempty"`                               // Created Epoch Timestamp
	LastModifiedOn           *int64                 `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn,omitempty" schema:"lastModifiedOn,omitempty"`                     // Last Modified Epoch Timestamp
	BudgetAmount             *float64               `json:"budgetAmount,omitempty" dynamodbav:"BudgetAmount,omitempty" schema:"budgetAmount,omitempty"`                         // Budget Amount allocated for this lease
	BudgetCurrency           *string                `json:"budgetCurrency,omitempty" dynamodbav:"BudgetCurrency,omitempty" schema:"budgetCurrency,omitempty"`                     // Budget currency
	BudgetNotificationEmails *[]string              `json:"budgetNotificationEmails,omitempty" dynamodbav:"BudgetNotificationEmails,omitempty" schema:"budgetNotificationEmails,omitempty"` // Budget notification emails
	StatusModifiedOn         *int64                 `json:"leaseStatusModifiedOn,omitempty" dynamodbav:"LeaseStatusModifiedOn,omitempty" schema:"leaseStatusModifiedOn,omitempty"`       // Last Modified Epoch Timestamp
	ExpiresOn                *int64                 `json:"expiresOn,omitempty" dynamodbav:"ExpiresOn,omitempty" schema:"expiresOn,omitempty"`                               // Lease expiration time as Epoch
	Limit                    *int64                 `json:"-" dynamodbav:"-" schema:"limit,omitempty"`
	NextAccountID            *string                `json:"-" dynamodbav:"-" schema:"nextAccountId,omitempty"`
	NextPrincipalID          *string                `json:"-" dynamodbav:"-" schema:"nextPrincipalId,omitempty"`
}

// Validate the lease data
func (l *Lease) Validate() error {
	err := validation.ValidateStruct(l,
		validation.Field(&l.ID, validateID...),
		validation.Field(&l.AccountID, validateAccountID...),
		validation.Field(&l.PrincipalID, validatePrincipalID...),
		validation.Field(&l.LastModifiedOn, validateInt64...),
		validation.Field(&l.Status, validateStatus...),
		validation.Field(&l.CreatedOn, validateInt64...),
	)
	if err != nil {
		return errors.NewValidation("lease", err)
	}
	return nil
}

// Leases is a list of type Lease
type Leases []Lease

// Status is a lease status type
type Status string

const (
	// StatusEmpty status
	StatusEmpty Status = ""
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
	return StatusEmpty, fmt.Errorf("Cannot parse value %s", status)
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
