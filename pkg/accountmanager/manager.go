package accountmanager

import (
	"fmt"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

// AccountManager manages account resources
type AccountManager struct {
	awsSession          *session.Session
	awsConfig           *aws.Config
	adminRoleArn        arn.ARN
	principalRoleName   string
	principalPolicyName string
	principalRoleArn    arn.ARN
	principalPolicyArn  arn.ARN
}

// Setup creates a new session manager struct
func (am *AccountManager) Setup(adminRoleArn arn.ARN) error {
	var err error
	// Prevent setting this multiple times just in case
	am.awsSession = session.Must(session.NewSession())

	// Create the credentials from AssumeRoleProvider to assume the role
	creds := stscreds.NewCredentials(am.awsSession, adminRoleArn.String())

	am.awsConfig = aws.NewConfig().WithCredentials(creds)

	am.adminRoleArn = adminRoleArn
	am.principalRoleArn, err = arn.Parse(
		fmt.Sprintf("arn:aws:iam::%s:policy/%s",
			am.adminRoleArn.AccountID, am.principalRoleName))
	if err != nil {
		return fmt.Errorf("Error creating Principal Role Arn: %s: %w", err, errors.ErrInternalServer)
	}
	am.principalPolicyArn, err = arn.Parse(
		fmt.Sprintf("arn:aws:iam::%s:policy/%s",
			am.adminRoleArn.AccountID, am.principalPolicyName))
	if err != nil {
		return fmt.Errorf("Error creating Principal Policy Arn: %s: %w", err, errors.ErrInternalServer)
	}

	return nil
}

// New creates a new account manager struct
func New(PrincipalRoleName string, PrincipalPolicyName string) (*AccountManager, error) {
	new := &AccountManager{
		principalRoleName:   PrincipalRoleName,
		principalPolicyName: PrincipalPolicyName,
	}

	return new, nil
}
