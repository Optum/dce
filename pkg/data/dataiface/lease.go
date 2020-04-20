//

package dataiface

import (
	"github.com/Optum/dce/pkg/lease"
)

// LeaseData makes working with the Lease Data Layer easier
type LeaseData interface {

	// Get the Lease record by ID
	Get(ID string) (*lease.Lease, error)

	// GetByAccountIDAndPrincipalID gets the Lease record by AccountID and PrincipalID
	GetByAccountIDAndPrincipalID(accountID string, principalID string) (*lease.Lease, error)

	List(query *lease.Lease) (*lease.Leases, error)
	// List Get a list of leases
	// Write the Lease record in DynamoDB
	// This is an upsert operation in which the record will either
	// be inserted or updated
	// prevLastModifiedOn parameter is the original lastModifiedOn
	Write(lease *lease.Lease, prevLastModifiedOn *int64) error
}
