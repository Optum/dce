//

package dataiface

import (
	"github.com/Optum/dce/pkg/model"
)

// AccountData makes working with the Account Data Layer easier
type AccountData interface {
	// WriteAccount the Account record in DynamoDB
	// This is an upsert operation in which the record will either
	// be inserted or updated
	// prevLastModifiedOn parameter is the original lastModifiedOn
	WriteAccount(account *model.Account, prevLastModifiedOn *int64) error
	// DeleteAccount the Account record in DynamoDB
	DeleteAccount(account *model.Account) error
	// GetAccountByID the Account record by ID
	GetAccountByID(accountID string) (*model.Account, error)
}
