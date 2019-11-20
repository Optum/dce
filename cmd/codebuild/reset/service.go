package main

import (
	"log"
	"os"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Declare singleton instances of each service
// See https://medium.com/golang-issue/how-singleton-pattern-works-with-golang-2fdd61cd5a7f
var (
	_config       *serviceConfig
	_awsSession   *session.Session
	_tokenService *common.STS
	_ssmService   *common.SSM
	_s3Service    *common.S3
	_snsService   *common.SNS
	_db           *db.DB
)

// service struct holds all the services to be used by
// the resetpipeline package
// Use `service.<serviceName>()` to retrieve singletons
// of each service
type service struct {
}

type serviceConfig struct {
	accountID                  string
	accountPrincipalRoleName   string
	accountPrincipalPolicyName string
	accountAdminRoleName       string
	accountAdminRoleARN        string
	allowedRegions             []string

	isNukeEnabled       bool
	nukeTemplateDefault string
	nukeTemplateBucket  string
	nukeTemplateKey     string
}

func (svc *service) config() *serviceConfig {
	if _config != nil {
		return _config
	}
	accountAdminRoleName := common.RequireEnv("RESET_ACCOUNT_ADMIN_ROLE_NAME")
	accountID := common.RequireEnv("RESET_ACCOUNT")
	_config = &serviceConfig{
		accountID:                  accountID,
		accountPrincipalRoleName:   common.RequireEnv("RESET_ACCOUNT_PRINCIPAL_ROLE_NAME"),
		accountPrincipalPolicyName: common.RequireEnv("RESET_ACCOUNT_PRINCIPAL_POLICY_NAME"),
		accountAdminRoleName:       accountAdminRoleName,
		accountAdminRoleARN:        "arn:aws:iam::" + accountID + ":role/" + accountAdminRoleName,

		isNukeEnabled:       os.Getenv("RESET_NUKE_TOGGLE") != "false",
		nukeTemplateDefault: common.RequireEnv("RESET_NUKE_TEMPLATE_DEFAULT"),
		nukeTemplateBucket:  common.RequireEnv("RESET_NUKE_TEMPLATE_BUCKET"),
		nukeTemplateKey:     common.RequireEnv("RESET_NUKE_TEMPLATE_KEY"),
	}
	return _config
}

// setConfig overrides the configuration used by the service struct.
// should only be used for testing
func (svc *service) setConfig(config *serviceConfig) {
	_config = config
}

func (svc *service) awsSession() *session.Session {
	if _awsSession != nil {
		return _awsSession
	}
	_awsSession, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	return _awsSession
}

func (svc *service) tokenService() *common.STS {
	if _tokenService != nil {
		return _tokenService
	}
	stsClient := sts.New(svc.awsSession())
	_tokenService = &common.STS{
		Client: stsClient,
	}
	return _tokenService
}

func (svc *service) ssmService() *common.SSM {
	if _ssmService != nil {
		return _ssmService
	}
	_ssmService = &common.SSM{
		Client: ssm.New(svc.awsSession()),
	}
	return _ssmService
}

func (svc *service) s3Service() *common.S3 {
	if _s3Service == nil {
		_s3Service = &common.S3{
			Client:  s3.New(svc.awsSession()),
			Manager: s3manager.NewDownloader(svc.awsSession()),
		}
	}
	return _s3Service
}

func (svc *service) db() *db.DB {
	if _db != nil {
		return _db
	}
	_db, err := db.NewFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize DB Service:  %s", err)
	}
	return _db
}

func (svc *service) snsService() *common.SNS {
	if _snsService == nil {
		_snsService = &common.SNS{
			Client: sns.New(svc.awsSession()),
		}
	}

	return _snsService
}
