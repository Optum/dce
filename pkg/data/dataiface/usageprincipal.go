//

package dataiface

// UsagePrincipal ...
type UsagePrincipal interface {
	// Write the Lease record in DynamoDB
	// This is an upsert operation in which the record will either
	// be inserted or updated
	// Write(usg *usage.Lease) (*usage.Lease, error)
}
