//

package eventiface

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/lease"
)

// Servicer makes work with the event Hub easier
type Servicer interface {
	// AccountCreate publish events
	AccountCreate(data *account.Account) error
	// AccountDelete publish events
	AccountDelete(data *account.Account) error
	// AccountUpdate publish events
	AccountUpdate(data *account.Account) error
	// AccountReset publish events
	AccountReset(data *account.Account) error
	// LeaseCreate publish events
	LeaseCreate(data *lease.Lease) error
	// LeaseEnd publish events
	LeaseEnd(i interface{}) error
	// LeaseUpdate publish events
	LeaseUpdate(i interface{}) error
}
