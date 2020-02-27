//

package usageiface

import "github.com/Optum/dce/pkg/usage"

// Servicer ...
type Servicer interface {
	// UpsertLeaseUsage creates a new lease usage record
	UpsertLeaseUsage(data *usage.Lease) error
	// GetLease gets a lease usage record
	GetLease(id string) (*usage.Lease, error)
	// ListLease returns usage lease usage records
	ListLease(query *usage.Lease) (*usage.Leases, error)
	// ListPrincipal returns usage lease usage records
	ListPrincipal(query *usage.Principal) (*usage.Principal, error)
}
