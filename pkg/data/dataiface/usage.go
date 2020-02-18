//

package dataiface

import (
	"github.com/Optum/dce/pkg/usage"
)

// UsageData makes working with the Usage Data Layer easier
type UsageData interface {

	// Get the usage records by startDate for input principalID
	Get(startDate int64, principalID string) (*usage.Usage, error)

	List(query *usage.Usage) (*usage.Usages, error)

	Write(usage *usage.Usage) error
}
