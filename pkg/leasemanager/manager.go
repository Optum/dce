package leasemanager

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

// LeaseManager manages lease resources
type LeaseManager struct {
	awsConfig *aws.Config
	sts       stsiface.STSAPI
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
	new := &LeaseManager{
		sts: sts.New(session.Must(session.NewSession())),
	}

	return new, nil
}
