//

package dataiface

import (
	"github.com/Optum/dce/pkg/usage"
)

// UsageData ...
type UsageData interface {
	// Write the Lease record in DynamoDB
	// This is an upsert operation in which the record will either
	// be inserted or updated
	Write(usg *usage.Usage) (*usage.Usage, error)
	// Add to CostAmount
	Add(usg *usage.Usage) (*usage.Usage, error)
}
