package accountmanager

import (
	"fmt"
	"strings"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	validation "github.com/go-ozzo/ozzo-validation"
)

// ServiceConfig has specific static values for the service configuration
type ServiceConfig struct {
	AccountID                   string   `env:"ACCOUNT_ID" envDefault:"111111111111"`
	S3BucketName                string   `env:"ARTIFACTS_BUCKET" envDefault:"DefaultArtifactBucket"`
	S3PolicyKey                 string   `env:"PRINCIPAL_POLICY_S3_KEY" envDefault:"DefaultPrincipalPolicyS3Key"`
	PrincipalIAMDenyTags        []string `env:"PRINCIPAL_IAM_DENY_TAGS" envDefault:"DefaultPrincipalIamDenyTags"`
	PrincipalMaxSessionDuration int64    `env:"PRINCIPAL_MAX_SESSION_DURATION" envDefault:"3600"` // 3600 is the default minimum value
	AllowedRegions              []string `env:"ALLOWED_REGIONS" envDefault:"us-east-1"`
	TagEnvironment              string   `env:"TAG_ENVIRONMENT" envDefault:"DefaultTagEnvironment"`
	TagContact                  string   `env:"TAG_CONTACT" envDefault:"DefaultTagContact"`
	TagAppName                  string   `env:"TAG_APP_NAME" envDefault:"DefaultTagAppName"`
	PrincipalRoleDescription    string   `env:"PRINCIPAL_ROLE_DESCRIPTION" envDefault:"Role for principal users of DCE"`
	PrincipalPolicyDescription  string   `env:"PRINCIPAL_POLICY_DESCRIPTION" envDefault:"Policy for principal users of DCE"`
	tags                        []*iam.Tag
	assumeRolePolicy            string
}

// Service manages account resources
type Service struct {
	client   clienter
	storager common.Storager
	config   ServiceConfig
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

// UpsertPrincipalAccess creates roles, policies and updates them as needed
func (s *Service) UpsertPrincipalAccess(account *account.Account) error {
	err := validation.ValidateStruct(account,
		validation.Field(&account.AdminRoleArn, validation.NotNil),
		validation.Field(&account.PrincipalRoleArn, validation.NotNil),
	)
	if err != nil {
		return errors.NewValidation("account", err)
	}

	iamSvc := s.client.IAM(account.AdminRoleArn)

	principalSvc := principalService{
		iamSvc:   iamSvc,
		storager: s.storager,
		account:  account,
		config:   s.config,
	}

	err = principalSvc.MergeRole()
	if err != nil {
		return err
	}
	err = principalSvc.MergePolicy()
	if err != nil {
		return err
	}

	err = principalSvc.AttachRoleWithPolicy()
	if err != nil {
		return err
	}

	return nil
}

// NewServiceInput are the items needed to create a new service
type NewServiceInput struct {
	Session  *session.Session
	Sts      stsiface.STSAPI
	Storager common.Storager
	Config   ServiceConfig
}

// NewService creates a new account manager server
func NewService(input NewServiceInput) (*Service, error) {

	new := &Service{
		client: &client{
			session: input.Session,
			sts:     input.Sts,
		},
		storager: input.Storager,
		config:   input.Config,
	}

	new.config.tags = []*iam.Tag{
		{Key: aws.String("Terraform"), Value: aws.String("False")},
		{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
		{Key: aws.String("Environment"), Value: aws.String(new.config.TagEnvironment)},
		{Key: aws.String("Contact"), Value: aws.String(new.config.TagContact)},
		{Key: aws.String("AppName"), Value: aws.String(new.config.TagAppName)},
	}

	new.config.assumeRolePolicy = strings.TrimSpace(fmt.Sprintf(`
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
	`, new.config.AccountID))

	return new, nil

}
