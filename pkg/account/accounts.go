package account

import (
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Accounts is a list of type Account
type Accounts struct {
	data []Data
}

// GetAccounts Get a list of accounts based on Principal ID
func GetAccounts(q *Account, d MultipleReader) (*Accounts, error) {
	err := validation.ValidateStruct(&q.data,
		// ID has to be empty
		validation.Field(&q.data.ID, validation.NilOrNotEmpty, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("account", err)
	}

	accounts := &Accounts{}
	err = d.GetAccounts(q, accounts)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}
