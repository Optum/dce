//

package accountmanageriface

import (
	"github.com/Optum/dce/pkg/account"
)

// Servicer makes working with the Account Manager easier
type Servicer interface {
	// ValidateAccess creates a new Account instance
	ValidateAccess(role string) error
	// MergePrincipalAccess creates roles, policies and update them as needed
	MergePrincipalAccess(account *account.Account) error
}
