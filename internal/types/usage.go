package types

import "time"

// Usages is used for a list of Usage information
type Usages []Usage

// Usage is an internal type for passing usage data from the cloud API
type Usage struct {
	TimePeriod time.Time
	Amount     float64
}
