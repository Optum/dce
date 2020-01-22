//

package dataiface

import (
	"github.com/Optum/dce/pkg/account"
)

// AccountData makes working with the Account Data Layer easier
type AccountData interface {
	// Write the Account record in DynamoDB
	// This is an upsert operation in which the record will either
	// be inserted or updated
	// prevLastModifiedOn parameter is the original lastModifiedOn
	Write(account *account.Account, prevLastModifiedOn *int64) error
	// Delete the Account record in DynamoDB
	Delete(account *account.Account) error
	// Get the Account record by ID
	Get(ID string) (*account.Account, error)
	// List Get a list of accounts
	List(query *account.Account) (*account.Accounts, error)
}
