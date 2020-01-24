//

package leasemanageriface

// LeaseManagerAPI makes working with the Lease Manager easier
type LeaseManagerAPI interface {
	// Setup creates a new lease manager struct
	Setup() error
}
