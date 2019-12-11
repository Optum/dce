//

package dataiface

import (
	"github.com/Optum/dce/pkg/model"
)

// AccountData makes working with the Account Data Layer easier
type AccountData interface {
	// Update the Account record in DynamoDB
	Update(account *model.Account, lastModifiedOn *int64) error
	// Delete the Account record in DynamoDB
	Delete(account *model.Account) error
	// GetAccountByID the Account record by ID
	GetAccountByID(accountID string, account *model.Account) error
	// GetAccountsByStatus - Returns the accounts by status
	GetAccountsByStatus(status string) (*model.Accounts, error)
	// GetAccountsByPrincipalID Get a list of accounts based on Principal ID
	GetAccountsByPrincipalID(principalID string) (*model.Accounts, error)
	// GetAccounts Get a list of accounts based on Principal ID
	GetAccounts() (*model.Accounts, error)
}
