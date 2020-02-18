//

package accountmanageriface

import (
	"time"

	"github.com/Optum/dce/internal/types"
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
)

// Servicer makes working with the Account Manager easier
type Servicer interface {
	// ValidateAccess creates a new Account instance
	ValidateAccess(role *arn.ARN) error
	// UpsertPrincipalAccess creates roles, policies and update them as needed
	UpsertPrincipalAccess(account *account.Account) error
	// DeletePrincipalAccess removes all the principal roles and policies
	DeletePrincipalAccess(account *account.Account) error
	// GetUsageBetweenDates gets the information from cost explorer
	GetUsageBetweenDates(account *account.Account, startDate time.Time, endDate time.Time) (types.Usages, error)
}
