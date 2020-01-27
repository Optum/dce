package leasemanager

// LeaseManager manages lease resources
type LeaseManager struct {
}

// Setup creates a new session manager struct
func (am *LeaseManager) Setup() error {

	return nil
}

// NewInput holds the configuration for a new LeaseManager
type NewInput struct {
	PrincipalRoleName   string `env:"PRINCIPAL_ROLE_NAME"`
	PrincipalPolicyName string `env:"PRINCIPAL_POLICY_NAME"`
}

// New creates a new lease manager struct
func New(input NewInput) (*LeaseManager, error) {
	new := &LeaseManager{}

	return new, nil
}
