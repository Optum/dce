//

package usageiface

import "github.com/Optum/dce/pkg/usage"

// Servicer ...
type Servicer interface {
	// UpsertLeaseUsage creates a new lease usage record
	UpsertLeaseUsage(data *usage.Lease) error
}
