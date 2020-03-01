//

package usageiface

import (
	"time"

	"github.com/Optum/dce/pkg/usage"
)

// Servicer ...
type Servicer interface {
	// UpsertLeaseUsage creates a new lease usage record
	UpsertLeaseUsage(data *usage.Lease) error
	// GetLease gets a lease usage record
	GetLease(id string) (*usage.Lease, error)
	// GetLease gets a lease usage record
	GetPrincipal(id string, date time.Time) (*usage.Principal, error)
	// ListLease returns usage lease usage records
	ListLease(query *usage.Lease) (*usage.Leases, error)
	// ListPrincipal returns usage lease usage records
	ListPrincipal(query *usage.Principal) (*usage.Principals, error)
}
