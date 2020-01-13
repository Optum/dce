package account

import (
	"encoding/json"

	"github.com/Optum/dce/pkg/model"
)

// Accounts is a list of type Account
type Accounts struct {
	data model.Accounts
}

// GetAccounts Get a list of accounts based on Principal ID
func GetAccounts(q *model.Account, d MultipleReader) (*Accounts, error) {

	accounts, err := d.GetAccounts(q)
	if err != nil {
		return nil, err
	}
	newAccounts := &Accounts{
		data: *accounts,
	}

	return newAccounts, nil
}

// MarshalJSON Marshals the data inside the account
func (a *Accounts) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.data)
}
