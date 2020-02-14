package accountmanager

import (
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/accountmanager/accountmanageriface"
	errors2 "github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

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

const (
	consoleURL    = "https://console.aws.amazon.com/"
	federationURL = "https://signin.aws.amazon.com/federation"
)

//go:generate mockery -name HTTPClienter
type HTTPClienter interface {
	Do(req *http.Request) (*http.Response, error)
}

// Service manages account resources
type Service struct {
	client     clienter
	storager   common.Storager
	config     ServiceConfig
	httpClient HTTPClienter
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

// Credentials assumes role into the provided ARN
// and returns back a wrapper object for retrieving STS crednetials
func (s *Service) Credentials(role *arn.ARN, roleSessionName string, duration *time.Duration) accountmanageriface.Credentialer {
	return s.client.Config(role, roleSessionName, duration).Credentials
}

// ConsoleURL generates a URL that may be used
// to login to the AWS web console for an account
func (s *Service) ConsoleURL(creds accountmanageriface.Credentialer) (string, error) {
	signinToken, err := s.signinToken(creds)
	if err != nil {
		return "", err
	}

	// have to use url.QueryEscape for the URL or its not properly escaped
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s?Destination=%s", federationURL, url.QueryEscape(consoleURL)),
		nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "DCE")
	q.Add("SigninToken", signinToken)
	req.URL.RawQuery = q.Encode()

	return req.URL.String(), nil
}

func (s Service) signinToken(creds accountmanageriface.Credentialer) (string, error) {
	// Retrieve the credentials
	credsValue, err := creds.Get()
	if err != nil {
		return "", err
	}
	credentialString, err := json.Marshal(map[string]string{
		"sessionId":    credsValue.AccessKeyID,
		"sessionKey":   credsValue.SecretAccessKey,
		"sessionToken": credsValue.SessionToken,
	})
	if err != nil {
		return "", errors2.Wrap(err, "Failed to marshall credentials for signin token")
	}

	req, err := http.NewRequest("GET", federationURL, nil)
	if err != nil {
		return "", errors2.Wrap(err, "Error building Request for signin token")
	}
	q := req.URL.Query()
	q.Add("Action", "getSigninToken")
	q.Add("Session", string(credentialString))
	req.URL.RawQuery = q.Encode()

	str := req.URL.String()
	qr := req.URL.Query()
	_ = str
	_ = qr

	resSigninToken, err := s.httpClient.Do(req)
	if err != nil {
		return "", errors2.Wrap(err, "Error getting signin token")
	}

	defer resSigninToken.Body.Close()
	bodySigninToken, err := ioutil.ReadAll(resSigninToken.Body)
	if err != nil {
		return "", errors2.Wrap(err, "Error getting signin token")
	}

	var signinToken struct {
		SigninToken string `json:"SigninToken"`
	}
	err = json.Unmarshal(bodySigninToken, &signinToken)
	if err != nil {
		return "", errors2.Wrap(err, "Error unmarshalling signin token repsonse")
	}
	return signinToken.SigninToken, nil
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

// DeletePrincipalAccess removes all the principal roles and policies
func (s *Service) DeletePrincipalAccess(account *account.Account) error {
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

	err = principalSvc.DetachRoleWithPolicy()
	if err != nil {
		return err
	}
	err = principalSvc.DeletePolicy()
	if err != nil {
		return err
	}

	err = principalSvc.DeleteRole()
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
		storager:   input.Storager,
		config:     input.Config,
		httpClient: &http.Client{},
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
