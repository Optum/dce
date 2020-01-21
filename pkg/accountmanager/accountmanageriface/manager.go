//

package accountmanageriface

// AccountManagerAPI makes working with the Account Manager easier
type AccountManagerAPI interface {
	// Setup creates a new session manager struct
	Setup(adminRoleArn string) error
}
