package usage

import (
	"time"

	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Principal item
type Principal struct {
	PrincipalID     *string    `json:"principalId,omitempty" dynamodbav:"PrincipalId" schema:"principalId,omitempty"`              // User Principal ID
	Date            *time.Time `json:"date,omitempty" dynamodbav:"Date,unixtime" schema:"date,omitempty"`                          // Usage date
	CostAmount      *float64   `json:"costAmount,omitempty" dynamodbav:"CostAmount,omitempty" schema:"costAmount,omitempty"`       // Cost Amount for given period
	CostCurrency    *string    `json:"costCurrency,omitempty" dynamodbav:"CostCurrency,omitempty" schema:"costCurrency,omitempty"` // Cost currency
	SK              *string    `json:"-" dynamodbav:"SK" schema:"-"`
	Limit           *int64     `json:"-" dynamodbav:"-" schema:"limit,omitempty"`
	NextDate        *int64     `json:"-" dynamodbav:"-" schema:"nextDate,omitempty"`
	NextPrincipalID *string    `json:"-" dynamodbav:"-" schema:"nextPrincipalId,omitempty"`
}

// Validate the account data
func (u *Principal) Validate() error {
	err := validation.ValidateStruct(u,
		validation.Field(&u.PrincipalID, validatePrincipalID...),
		validation.Field(&u.CostAmount, validateFloat64...),
		validation.Field(&u.CostCurrency, validateCostCurrency...),
	)
	if err != nil {
		return errors.NewValidation("usage", err)
	}
	return nil
}

// NewPrincipalInput has the input for create a new principal usage record
type NewPrincipalInput struct {
	PrincipalID  string
	Date         time.Time
	CostAmount   float64
	CostCurrency string
}

// NewPrincipal creates a new instance of usage
func NewPrincipal(input NewPrincipalInput) (*Principal, error) {

	new := &Principal{
		PrincipalID:  &input.PrincipalID,
		Date:         &input.Date,
		CostAmount:   &input.CostAmount,
		CostCurrency: &input.CostCurrency,
	}

	err := new.Validate()
	if err != nil {
		return nil, err
	}
	return new, nil

}

// Principals is a list of Principal Usages
type Principals []Principal
