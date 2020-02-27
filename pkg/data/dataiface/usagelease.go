//

package dataiface

import (
	"github.com/Optum/dce/pkg/usage"
)

// UsageLease ...
type UsageLease interface {
	// Write the Lease record in DynamoDB
	// This is an upsert operation in which the record will either
	// be inserted or updated
	Write(usg *usage.Lease) error
	List(query *usage.Lease) (*usage.Leases, error)
	Get(id string) (*usage.Lease, error)
}
