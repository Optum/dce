package account

import (
	"github.com/Optum/dce/pkg/model"
)

type Accounts []Account

func modelToAccounts(accounts *model.Accounts) *Accounts {
	res := Accounts{}
	for _, a := range *accounts {
		res = append(res, Account{
			data: a,
		})
	}
	return &res
}

// GetAccountsByStatus - Returns the accounts by status
func GetAccountsByStatus(status model.AccountStatus, d MultipleReader) (*Accounts, error) {
	accounts := &model.Accounts{}
	accounts, err := d.GetAccountsByStatus(string(status))
	if err != nil {
		return nil, err
	}

	return modelToAccounts(accounts), nil
}

// GetAccountsByPrincipalID Get a list of accounts based on Principal ID
func GetAccountsByPrincipalID(principalID string, d MultipleReader) (*Accounts, error) {
	accounts := &model.Accounts{}
	accounts, err := d.GetAccountsByPrincipalID(principalID)
	if err != nil {
		return nil, err
	}

	return modelToAccounts(accounts), nil
}

// GetAccounts Get a list of accounts based on Principal ID
func GetAccounts(d MultipleReader) (*Accounts, error) {
	accounts := &model.Accounts{}
	accounts, err := d.GetAccounts()
	if err != nil {
		return nil, err
	}

	return modelToAccounts(accounts), nil
}
