package accountmanager

import (
	"fmt"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

// AccountManager manages account resources
type AccountManager struct {
	awsConfig           *aws.Config
	sts                 stsiface.STSAPI
	adminRoleArn        arn.ARN
	principalRoleName   string `env:"PRINCIPAL_ROLE_NAME"`
	principalPolicyName string `env:"PRINCIPAL_POLICY_NAME"`
	principalRoleArn    arn.ARN
	principalPolicyArn  arn.ARN
}

// Setup creates a new session manager struct
func (am *AccountManager) Setup(adminRoleArn string) error {
	var err error

	am.adminRoleArn, err = arn.Parse(adminRoleArn)
	if err != nil {
		return errors.NewValidation("admin role arn", err)
	}

	// Create the credentials from AssumeRoleProvider to assume the role
	creds := stscreds.NewCredentialsWithClient(am.sts, am.adminRoleArn.String())
	_, err = creds.Get()
	if err != nil {
		return errors.NewValidation("admin role arn", err)
	}
	am.awsConfig = aws.NewConfig().WithCredentials(creds)

	am.principalRoleArn, err = arn.Parse(
		fmt.Sprintf("arn:aws:iam::%s:policy/%s",
			am.adminRoleArn.AccountID, am.principalRoleName))
	if err != nil {
		return errors.NewInternalServer("error creating principal role arn", err)
	}
	am.principalPolicyArn, err = arn.Parse(
		fmt.Sprintf("arn:aws:iam::%s:policy/%s",
			am.adminRoleArn.AccountID, am.principalPolicyName))
	if err != nil {
		return errors.NewInternalServer("error creating principal policy arn", err)
	}

	return nil
}

// NewInput holds the configuration for a new AccountManager
type NewInput struct {
	PrincipalRoleName   string `env:"PRINCIPAL_ROLE_NAME"`
	PrincipalPolicyName string `env:"PRINCIPAL_POLICY_NAME"`
}

// New creates a new account manager struct
func New(input NewInput) (*AccountManager, error) {
	new := &AccountManager{
		principalRoleName:   input.PrincipalRoleName,
		principalPolicyName: input.PrincipalPolicyName,
		sts:                 sts.New(session.Must(session.NewSession())),
	}

	return new, nil
}
