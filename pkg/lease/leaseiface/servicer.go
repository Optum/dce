//

package leaseiface

import (
	"github.com/Optum/dce/pkg/lease"
)

// Servicer makes working with the Lease Service struct easier
type Servicer interface {
	// Get returns an lease from ID
	Get(ID string) (*lease.Lease, error)

	//// Save writes the record to the dataSvc
	//Save(data *lease.Lease) error

	// Update the Lease record to status Inactive in DynamoDB
	Delete(ID string) (*lease.Lease, error)

	// List Get a list of lease based on Lease ID
	List(query *lease.Lease) (*lease.Leases, error)
}
