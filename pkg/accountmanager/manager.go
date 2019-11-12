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
func (am *AccountManager) Setup(i string) error {
	var err error

	adminRoleArn, err := arn.Parse(i)
	if err != nil {
		return &errors.ErrValidation{
			Message: "admin role arn is not a valid arn",
			Err:     err,
		}
	}

	// Prevent setting this multiple times just in case
	am.awsSession = session.Must(session.NewSession())

	// Create the credentials from AssumeRoleProvider to assume the role
	creds := stscreds.NewCredentials(am.awsSession, adminRoleArn.String())
	_, err = creds.Get()
	if err != nil {
		return &errors.ErrValidation{
			Message: "cannot assume admin role arn",
			Err:     err,
		}
	}
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
