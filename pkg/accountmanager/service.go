package accountmanager

import (
	"fmt"
	"log"
	"strings"

	"github.com/caarlos0/env/v6"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	validation "github.com/go-ozzo/ozzo-validation"
)

type serviceConfig struct {
	AccountID                   string   `env:"ACCOUNT_ID" envDefault:"111111111111"`
	S3BucketName                string   `env:"ARTIFACTS_BUCKET" envDefault:"DefaultArtifactBucket"`
	S3PolicyKey                 string   `env:"PRINCIPAL_POLICY_S3_KEY" envDefault:"DefaultPrincipalPolicyS3Key"`
	PrincipalIAMDenyTags        []string `env:"PRINCIPAL_IAM_DENY_TAGS" envDefault:"DefaultPrincipalIamDenyTags"`
	PrincipalMaxSessionDuration int64    `env:"PRINCIPAL_MAX_SESSION_DURATION" envDefault:"3600"` // 3600 is the default minimum value
	AllowedRegions              []string `env:"ALLOWED_REGIONS" envDefault:"us-east-1"`
	TagEnvironment              string   `env:"TAG_ENVIRONMENT" envDefault:"DefaultTagEnvironment"`
	TagContact                  string   `env:"TAG_CONTACT" envDefault:"DefaultTagContact"`
	TagAppName                  string   `env:"TAG_APP_NAME" envDefault:"DefaultTagAppName"`
}

var (
	// Config holds static configuration values
	Config serviceConfig
	// Tags has the default IAM Tags
	Tags []*iam.Tag
	// AssumeRolePolicy Default Assume Role Policy
	AssumeRolePolicy string
)

func init() {
	if err := env.Parse(&Config); err != nil {
		panic(err)
	}

	Tags = []*iam.Tag{
		{Key: aws.String("Terraform"), Value: aws.String("False")},
		{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
		{Key: aws.String("Environment"), Value: aws.String(Config.TagEnvironment)},
		{Key: aws.String("Contact"), Value: aws.String(Config.TagContact)},
		{Key: aws.String("AppName"), Value: aws.String(Config.TagAppName)},
	}

	AssumeRolePolicy = strings.TrimSpace(fmt.Sprintf(`
		{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": "arn:aws:iam::%s:root"
					},
					"Action": "sts:AssumeRole",
					"Condition": {}
				}
			]
		}
	`, Config.AccountID))

}

// Service manages account resources
type Service struct {
	client   clienter
	sts      stsiface.STSAPI
	storager common.Storager
}

// ValidateAccess creates a new Account instance
func (s *Service) ValidateAccess(role *arn.ARN) error {
	err := validation.Validate(role,
		validation.NotNil,
		validation.By(isAssumable(s.client)))
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

	iamSvc := s.client.IAM(account.AdminRoleArn)

	err = s.mergePrincipalRole(iamSvc, account)
	if err != nil {
		return err
	}
	err = s.mergePrincipalPolicy(iamSvc, account)
	if err != nil {
		return err
	}

	err = s.mergePrincipalRoleWithPolicy(iamSvc, account)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) mergePrincipalRole(iamSvc iamiface.IAMAPI, account *account.Account) error {

	_, err := iamSvc.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(*account.PrincipalRoleName),
		AssumeRolePolicyDocument: aws.String(AssumeRolePolicy),
		Description:              aws.String("Role to be assumed by principal users of DCE"),
		MaxSessionDuration:       aws.Int64(Config.PrincipalMaxSessionDuration),
		Tags: append(Tags,
			&iam.Tag{Key: aws.String("Name"), Value: aws.String("DCEPrincipal")},
		),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return err
		}
	}

	return nil
}

func (s *Service) mergePrincipalPolicy(iamSvc iamiface.IAMAPI, account *account.Account) error {

	policy, policyHash, err := s.buildPolicy(account)
	if err != nil {
		return err
	}

	if policyHash != account.PrincipalPolicyHash {
		err = mergePolicy(&mergePolicyInput{
			iam:         iamSvc,
			policyArn:   account.PrincipalPolicyArn.ARN,
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

func (s *Service) mergePrincipalRoleWithPolicy(iamSvc iamiface.IAMAPI, account *account.Account) error {

	// Attach the policy to the role
	_, err := iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: aws.String(account.PrincipalPolicyArn.String()),
		RoleName:  aws.String(*account.PrincipalRoleName),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return err
		}
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

	policy, policyHash, err := s.storager.GetTemplateObject(Config.S3BucketName, Config.S3PolicyKey,
		principalPolicyInput{
			PrincipalPolicyArn:   account.PrincipalPolicyArn.String(),
			PrincipalRoleArn:     account.PrincipalRoleArn.String(),
			PrincipalIAMDenyTags: Config.PrincipalIAMDenyTags,
			AdminRoleArn:         account.AdminRoleArn.String(),
			Regions:              Config.AllowedRegions,
		})
	if err != nil {
		return nil, nil, err
	}

	return &policy, &policyHash, nil
}

// NewServiceInput are the items needed to create a new service
type NewServiceInput struct {
	Session  *session.Session
	Sts      stsiface.STSAPI
	Storager common.Storager
}

// NewService creates a new account manager server
func NewService(input NewServiceInput) (*Service, error) {

	return &Service{
		client: &client{
			session: input.Session,
			sts:     input.Sts,
		},
		storager: input.Storager,
	}, nil

}
