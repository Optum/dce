package usage

import (
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Usage item
type Usage struct {
	PrincipalID  *string  `json:"principalId,omitempty" dynamodbav:"PrincipalId" schema:"principalId,omitempty"`              // User Principal ID
	AccountID    *string  `json:"accountId,omitempty" dynamodbav:"AccountId,omitempty" schema:"accountId,omitempty"`          // AWS Account ID
	StartDate    *int64   `json:"startDate,omitempty" dynamodbav:"StartDate" schema:"startDate,omitempty"`                    // Usage start date Epoch Timestamp
	EndDate      *int64   `json:"endDate,omitempty" dynamodbav:"EndDate,omitempty" schema:"endDate,omitempty"`                // Usage ends date Epoch Timestamp
	CostAmount   *float64 `json:"costAmount,omitempty" dynamodbav:"CostAmount,omitempty" schema:"costAmount,omitempty"`       // Cost Amount for given period
	CostCurrency *string  `json:"costCurrency,omitempty" dynamodbav:"CostCurrency,omitempty" schema:"costCurrency,omitempty"` // Cost currency
	TimeToLive   *int64   `json:"timeToLive,omitempty" dynamodbav:"TimeToLive,omitempty" schema:"timeToLive,omitempty"`       // ttl attribute
}

// Validate the account data
func (u *Usage) Validate() error {
	err := validation.ValidateStruct(u,
		validation.Field(&u.PrincipalID, validatePrincipalID...),
		validation.Field(&u.AccountID, validateAccountID...),
		validation.Field(&u.StartDate, validateInt64...),
		validation.Field(&u.EndDate, validateInt64...),
		validation.Field(&u.CostAmount, validateFloat64...),
		validation.Field(&u.CostCurrency, validateCostCurrency...),
		validation.Field(&u.TimeToLive, validateTimeToLive...),
	)
	if err != nil {
		return errors.NewValidation("usage", err)
	}
	return nil
}

// NewUsageInput has the input for create a new usage record
type NewUsageInput struct {
	PrincipalID  string
	AccountID    string
	StartDate    int64
	EndDate      int64
	CostAmount   float64
	CostCurrency string
	TimeToLive   int64
}

// NewUsage creates a new instance of usage
func NewUsage(input NewUsageInput) (*Usage, error) {

	new := &Usage{
		PrincipalID:  &input.PrincipalID,
		AccountID:    &input.AccountID,
		StartDate:    &input.StartDate,
		EndDate:      &input.EndDate,
		CostAmount:   &input.CostAmount,
		CostCurrency: &input.CostCurrency,
		TimeToLive:   &input.TimeToLive,
	}

	err := new.Validate()
	if err != nil {
		return nil, err
	}
	return new, nil

}

// Usages is a list of type Usage
type Usages []Usage
