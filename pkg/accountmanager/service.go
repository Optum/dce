package accountmanager

import (
	"fmt"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Service manages account resources
type Service struct {
	Session                     *session.Session
	Storager                    common.Storager
	S3BucketName                string   `env:"ARTIFACTS_BUCKET" defaultEnv:"DefaultArtifactBucket"`
	S3PolicyKey                 string   `env:"PRINCIPAL_POLICY_S3_KEY" defaultEnv:"DefaultPrincipalPolicyS3Key"`
	PrincipalRoleName           string   `env:"PRINCIPAL_ROLE_NAME" defaultEnv:"DCEPrincipal"`
	PrincipalPolicyName         string   `env:"PRINCIPAL_POLICY_NAME" defaultEnv:"DCEPrincipalDefaultPolicy"`
	PrincipalIAMDenyTags        []string `env:"PRINCIPAL_IAM_DENY_TAGS" defaultEnv:"DefaultPrincipalIamDenyTags"`
	PrincipalMaxSessionDuration int64    `env:"PRINCIPAL_MAX_SESSION_DURATION" defaultEnv:"100"`
	AllowedRegions              []string `env:"ALLOWED_REGIONS" defaultEnv:"us-east-1"`
}

// ValidateAccess creates a new Account instance
func (s *Service) ValidateAccess(role arn.ARN) error {
	err := validation.Validate(role,
		validation.NotNil,
		validation.By(isAssumable(s.Session)))
	if err != nil {
		return errors.NewValidation("account", err)
	}
	return nil
}

// MergePrincipalAccess creates roles, policies and update them as needed
func (s *Service) MergePrincipalAccess(account *account.Account) error {
	err := validation.ValidateStruct(account,
		validation.Field(&account.AdminRoleArn, validation.NotNil),
		validation.Field(&account.PrincipalRoleArn, validation.NotNil),
	)
	if err != nil {
		return errors.NewValidation("account", err)
	}

	// Build the Policy ARN - Nothing else does this for us
	policyArn, err := arn.Parse(fmt.Sprintf("arn:aws:iam::%s:policy/%s", *account.ID, s.PrincipalPolicyName))
	if err != nil {
		return errors.NewInternalServer("error parsing policy arn", err)
	}

	creds := stscreds.NewCredentials(s.Session, *account.AdminRoleArn)
	iamSvc := iam.New(s.Session, &aws.Config{Credentials: creds})

	err = s.mergePrincipalRole(iamSvc, account)
	err = s.mergePrincipalPolicy(iamSvc, account, policyArn)
	return nil
}

func (s *Service) mergePrincipalRole(iamSvc iamiface.IAMAPI, account *account.Account) error {

	return nil
}

func (s *Service) mergePrincipalPolicy(iamSvc iamiface.IAMAPI, account *account.Account, policyArn arn.ARN) error {

	policy, policyHash, err := s.buildPolicy(account)
	if err != nil {
		return err
	}

	if policyHash != account.PrincipalPolicyHash {
		err = mergePolicy(&mergePolicyInput{
			iam:         nil,
			policyArn:   policyArn,
			description: aws.String("Policy for principal users of DCE"),
			document:    aws.String(*policy),
		})
		if err != nil {
			return err
		}

		account.PrincipalPolicyHash = policyHash
	}

	return nil
}

func (s *Service) buildPolicy(account *account.Account) (*string, *string, error) {

	type principalPolicyInput struct {
		PrincipalPolicyArn   string
		PrincipalRoleArn     string
		PrincipalIAMDenyTags []string
		AdminRoleArn         string
		Regions              []string
	}

	policy, policyHash, err := s.Storager.GetTemplateObject(s.S3BucketName, s.S3PolicyKey,
		principalPolicyInput{
			PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", *account.ID, s.PrincipalPolicyName),
			PrincipalRoleArn:     *account.PrincipalRoleArn,
			PrincipalIAMDenyTags: s.PrincipalIAMDenyTags,
			AdminRoleArn:         *account.AdminRoleArn,
			Regions:              s.AllowedRegions,
		})
	if err != nil {
		return nil, nil, err
	}

	return &policy, &policyHash, nil
}
