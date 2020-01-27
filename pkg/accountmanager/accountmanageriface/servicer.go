//

package accountmanageriface

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
)

// Servicer makes working with the Account Manager easier
type Servicer interface {
	// ValidateAccess creates a new Account instance
	ValidateAccess(role *arn.ARN) error
	// MergePrincipalAccess creates roles, policies and update them as needed
	MergePrincipalAccess(account *account.Account) error
}
