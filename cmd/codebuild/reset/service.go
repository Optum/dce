package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/reset"
)

// Declare singleton instances of each service
// See https://medium.com/golang-issue/how-singleton-pattern-works-with-golang-2fdd61cd5a7f
var (
	_config       *serviceConfig
	_launchpadAPI *reset.LaunchpadAPI
	_awsSession   *session.Session
	_tokenService *common.STS
	_ssmService   *common.SSM
	_db           *db.DB
)

// service struct holds all the services to be used by
// the resetpipeline package
// Use `service.<serviceName>()` to retrieve singletons
// of each service
type service struct {
}

type serviceConfig struct {
	accountID        string
	customerRoleName string
	customerRoleArn  string

	isNukeEnabled bool
	nukeTemplate  string

	isLaunchpadEnabled     bool
	launchpadBaseEndpoint  string
	launchpadAuthEndpoint  string
	launchpadMasterAccount string
	launchpadBackend       string
}

func (svc *service) config() *serviceConfig {
	if _config != nil {
		return _config
	}
	resetRole := common.RequireEnv("RESET_ROLE")
	accountID := common.RequireEnv("RESET_ACCOUNT")
	_config = &serviceConfig{
		accountID:        accountID,
		customerRoleName: resetRole,
		customerRoleArn:  "arn:aws:iam::" + accountID + ":role/" + resetRole,

		nukeTemplate:  common.RequireEnv("RESET_TEMPLATE"),
		isNukeEnabled: os.Getenv("RESET_NUKE_TOGGLE") != "false",

		isLaunchpadEnabled:     os.Getenv("RESET_LAUNCHPAD_TOGGLE") != "false",
		launchpadBaseEndpoint:  common.RequireEnv("RESET_LAUNCHPAD_BASE_ENDPOINT"),
		launchpadAuthEndpoint:  common.RequireEnv("RESET_LAUNCHPAD_AUTH_ENDPOINT"),
		launchpadMasterAccount: common.RequireEnv("RESET_LAUNCHPAD_MASTER_ACCOUNT"),
		launchpadBackend:       common.RequireEnv("RESET_LAUNCHPAD_BACKEND"),
	}
	return _config
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

func (svc *service) launchpadAPI() *reset.LaunchpadAPI {
	if _launchpadAPI != nil {
		return _launchpadAPI
	}
	config := svc.config()

	keyID := "/redbox/azure/client/id"
	keySecret := "/redbox/azure/client/secret"

	ssmService := svc.ssmService()
	clientID, err := ssmService.GetParameter(&keyID)
	if err != nil {
		log.Fatalf("%s  :  %s\n", config.accountID, err)
	}
	clientSecret, err := ssmService.GetParameter(&keySecret)
	if err != nil {
		log.Fatalf("%s  :  %s\n", config.accountID, err)
	}

	// Create the Storage service under the assumed role
	awsSession := svc.awsSession()
	tokenService := svc.tokenService()
	creds := tokenService.NewCredentials(awsSession, config.customerRoleArn)
	s3Client := s3.New(awsSession, &aws.Config{
		Credentials: creds,
	})
	storage := common.S3{
		Client: s3Client,
	}

	// Create the HTTPClient that will make the requests
	httpClient := common.HTTPClient{
		Client: http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	_launchpadAPI = &reset.LaunchpadAPI{
		LaunchpadBaseEndpoint: config.launchpadBaseEndpoint,
		LaunchpadAuthEndpoint: config.launchpadAuthEndpoint,
		ClientID:              *clientID,
		ClientSecret:          *clientSecret,
		BackendBucket:         config.launchpadBackend,
		HTTP:                  &httpClient,
		Storage:               storage,
		Token:                 tokenService,
	}

	return _launchpadAPI
}

func (svc *service) db() *db.DB {
	if _db != nil {
		return _db
	}
	_db, err := db.NewFromEnv()
	if err != nil {
		log.Fatalf("%s  :  %s\n", svc.config().accountID, err)
	}
	return _db
}
