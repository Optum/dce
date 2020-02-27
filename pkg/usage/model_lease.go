package usage

import (
	"time"

	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Lease item
type Lease struct {
	PrincipalID     *string    `json:"principalId,omitempty" dynamodbav:"PrincipalId" schema:"principalId,omitempty"`              // User Principal ID
	LeaseID         *string    `json:"leaseId,omitempty" dynamodbav:"LeaseId,omitempty" schema:"leaseId,omitempty"`                // Lease ID
	Date            *time.Time `json:"date,omitempty" dynamodbav:"Date,unixtime" schema:"date,omitempty"`                          // Usage date
	CostAmount      *float64   `json:"costAmount,omitempty" dynamodbav:"CostAmount,omitempty" schema:"costAmount,omitempty"`       // Cost Amount for given period
	CostCurrency    *string    `json:"costCurrency,omitempty" dynamodbav:"CostCurrency,omitempty" schema:"costCurrency,omitempty"` // Cost currency
	BudgetAmount    *float64   `json:"budgetAmount" dynamodbav:"BudgetAmount,omitempty" schema:"budgetAmount,omitempty"`
	TTL             *int64     `json:"-,omitempty" dynamodbav:"TimeToLive,omitempty" schema:"-,omitempty"`
	Limit           *int64     `json:"-" dynamodbav:"-" schema:"limit,omitempty"`
	NextDate        *int64     `json:"-" dynamodbav:"-" schema:"nestDate,omitempty"`
	NextPrincipalID *string    `json:"-" dynamodbav:"-" schema:"nextPrincipalId,omitempty"`
	NextLeaseID     *string    `json:"-" dynamodbav:"-" schema:"nextLeaseId,omitempty"`
}

// Validate the account data
func (u *Lease) Validate() error {
	err := validation.ValidateStruct(u,
		validation.Field(&u.PrincipalID, validatePrincipalID...),
		validation.Field(&u.LeaseID, validationLeaseID...),
		validation.Field(&u.CostAmount, validateFloat64...),
		validation.Field(&u.CostCurrency, validateCostCurrency...),
		validation.Field(&u.Date, validateInt64...),
		validation.Field(&u.TTL, validateTimeToLive...),
	)
	if err != nil {
		return errors.NewValidation("usage", err)
	}
	return nil
}

// NewLeaseInput has the input for create a new usage record
type NewLeaseInput struct {
	PrincipalID  string
	LeaseID      string
	Date         time.Time
	CostAmount   float64
	CostCurrency string
	BudgetAmount float64
}

// NewLease creates a new instance of usage
func NewLease(input NewLeaseInput) (*Lease, error) {

	new := &Lease{
		PrincipalID:  &input.PrincipalID,
		LeaseID:      &input.LeaseID,
		Date:         &input.Date,
		CostAmount:   &input.CostAmount,
		CostCurrency: &input.CostCurrency,
		BudgetAmount: &input.BudgetAmount,
	}

	err := new.Validate()
	if err != nil {
		return nil, err
	}
	return new, nil

}

// Leases is a list of lease usages
type Leases []Lease
