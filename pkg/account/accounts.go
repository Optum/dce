package account

import (
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Accounts is a list of type Account
type Accounts struct {
	data model.Accounts
}

// GetAccounts Get a list of accounts based on Principal ID
func GetAccounts(q *model.Account, d MultipleReader) (*Accounts, error) {
	err := validation.ValidateStruct(q,
		// ID has to be empty
		validation.Field(&q.ID, validation.NilOrNotEmpty, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("account", err)
	}

	newAccounts := &Accounts{
		data: model.Accounts{},
	}
	accounts, err := d.GetAccounts(q)
	if err != nil {
		return nil, err
	}
	newAccounts.data = *accounts

	return newAccounts, nil
}
