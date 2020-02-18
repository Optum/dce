//

package usageiface

import (
	"github.com/Optum/dce/pkg/usage"
)

// Servicer makes working with the Usage Service struct easier
type Servicer interface {
	// Get returns usage from startDate for input principalID
	Get(startDate int64, principalID string) (*usage.Usages, error)
}
