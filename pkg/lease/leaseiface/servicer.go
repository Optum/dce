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
	//// Update the Lease record in DynamoDB
	//Update(ID string, data *lease.Lease) (*lease.Lease, error)
	//// Delete finds a given lease and deletes it if it is not of status `Leased`. Returns the lease.
	//Delete(data *lease.Lease) error
	//// List Get a list of lease based on Account ID
	//List(query *lease.Lease) (*lease.Leases, error)
}
