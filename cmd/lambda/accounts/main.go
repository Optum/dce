package main

import (
	"context"
	"encoding/json"
	gErrors "errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/Optum/dce/pkg/config"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

// Accounter - Only exported for mockery
type Accounter interface {
	Update(d model.Account, u account.Updater, am account.Manager) error
}

type accountControllerConfiguration struct {
	Debug                       string   `env:"DEBUG" defaultEnv:"false"`
	PolicyName                  string   `env:"PRINCIPAL_POLICY_NAME" defaultEnv:"DCEPrincipalDefaultPolicy"`
	AccountCreatedTopicArn      string   `env:"ACCOUNT_CREATED_TOPIC_ARN" defaultEnv:"DefaultAccountCreatedTopicArn"`
	AccountDeletedTopicArn      string   `env:"ACCOUNT_DELETED_TOPIC_ARN"`
	ArtifactsBucket             string   `env:"ARTIFACTS_BUCKET" defaultEnv:"DefaultArtifactBucket"`
	PrincipalPolicyS3Key        string   `env:"PRINCIPAL_POLICY_S3_KEY" defaultEnv:"DefaultPrincipalPolicyS3Key"`
	PrincipalRoleName           string   `env:"PRINCIPAL_ROLE_NAME" defaultEnv:"DCEPrincipal"`
	PrincipalPolicyName         string   `env:"PRINCIPAL_POLICY_NAME"`
	PrincipalIAMDenyTags        []string `env:"PRINCIPAL_IAM_DENY_TAGS" defaultEnv:"DefaultPrincipalIamDenyTags"`
	PrincipalMaxSessionDuration int64    `env:"PRINCIPAL_MAX_SESSION_DURATION" defaultEnv:"100"`
	Tags                        []*iam.Tag
	ResetQueueURL               string   `env:"RESET_SQS_URL" defaultEnv:"DefaultResetSQSUrl"`
	AllowedRegions              []string `env:"ALLOWED_REGIONS" defaultEnv:"us-east-1"`
}

type tagSettings struct {
	environment string `env:"TAG_ENVIRONMENT" defaultEnv:"DefaultTagEnvironment"`
	contact     string `env:"TAG_CONTACT" defaultEnv:"DefaultTagContact"`
	appName     string `env:"TAG_APP_NAME" defaultEnv:"DefaultTagAppName"`
}

var (
	muxLambda *gorillamux.GorillaMuxAdapter
	//CurrentAccountID is the ID where the request is being created
	CurrentAccountID *string
	// Services handles the configuration of the AWS services
	Services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	Settings *accountControllerConfiguration
)

var (
	// Soon to be deprecated - Legacy support
	AWSSession  *session.Session
	Dao         db.DBer
	SnsSvc      common.Notificationer
	Queue       common.Queue
	TokenSvc    common.TokenService
	StorageSvc  common.Storager
	RoleManager rolemanager.RoleManager
	Config      common.DefaultEnvConfig
)

var (
	accountCreatedTopicArn      string
	policyName                  string
	artifactsBucket             string
	principalPolicyS3Key        string
	principalRoleName           string
	principalIAMDenyTags        []string
	principalMaxSessionDuration int64
	tags                        []*iam.Tag
	resetQueueURL               string
	allowedRegions              []string
)

func init() {
	initConfig()

	log.Println("Cold start; creating router for /accounts")
	accountRoutes := api.Routes{
		// Routes with query strings always go first,
		// because the matcher will stop on the first match
		api.Route{
			"GetAccountByStatus",
			"GET",
			"/accounts",
			[]string{"accountStatus"},
			GetAccountByStatus,
		},

		// Routes without query strings go after all of the
		// routes that use query strings for matchers.
		api.Route{
			"GetAllAccounts",
			"GET",
			"/accounts",
			api.EmptyQueryString,
			GetAccounts,
		},
		api.Route{
			"GetAccountByID",
			"GET",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			GetAccountByID,
		},
		api.Route{
			"UpdateAccountByID",
			"PUT",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			UpdateAccountByID,
		},
		api.Route{
			"DeleteAccount",
			"DELETE",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			DeleteAccount,
		},
		api.Route{
			"CreateAccount",
			"POST",
			"/accounts",
			api.EmptyQueryString,
			CreateAccount,
		},
	}
	r := api.NewRouter(Services.Config, accountRoutes)
	muxLambda = gorillamux.New(r)
}

// initConfig configures package-level variables
// loaded from env vars.
func initConfig() {
	cfgBldr := &config.ConfigurationBuilder{}
	Settings = &accountControllerConfiguration{}
	if err := cfgBldr.Unmarshal(Settings); err != nil {
		log.Fatalf("Could not load configuration: %s", err.Error())
	}

	// load up the values into the various settings...
	cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()
	svcBldr := &config.ServiceBuilder{Config: cfgBldr}

	_, err := svcBldr.
		// AWS services...
		WithDynamoDB().
		WithSTS().
		WithS3().
		WithSNS().
		WithSQS().
		// DCE services...
		WithDAO().
		WithRoleManager().
		WithStorageService().
		WithDataService().
		WithAccountManager().
		Build()

	if err == nil {
		Services = svcBldr
	}

	policyName = Config.GetEnvVar("PRINCIPAL_POLICY_NAME", "DCEPrincipalDefaultPolicy")
	artifactsBucket = Config.GetEnvVar("ARTIFACTS_BUCKET", "DefaultArtifactBucket")
	principalPolicyS3Key = Config.GetEnvVar("PRINCIPAL_POLICY_S3_KEY", "DefaultPrincipalPolicyS3Key")
	principalRoleName = Config.GetEnvVar("PRINCIPAL_ROLE_NAME", "DCEPrincipal")
	principalIAMDenyTags = strings.Split(Config.GetEnvVar("PRINCIPAL_IAM_DENY_TAGS", "DefaultPrincipalIamDenyTags"), ",")
	principalMaxSessionDuration = int64(Config.GetEnvIntVar("PRINCIPAL_MAX_SESSION_DURATION", 100))
	tags = []*iam.Tag{
		{Key: aws.String("Terraform"), Value: aws.String("False")},
		{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
		{Key: aws.String("Environment"), Value: aws.String(Config.GetEnvVar("TAG_ENVIRONMENT", "DefaultTagEnvironment"))},
		{Key: aws.String("Contact"), Value: aws.String(Config.GetEnvVar("TAG_CONTACT", "DefaultTagContact"))},
		{Key: aws.String("AppName"), Value: aws.String(Config.GetEnvVar("TAG_APP_NAME", "DefaultTagAppName"))},
	}
	accountCreatedTopicArn = Config.GetEnvVar("ACCOUNT_CREATED_TOPIC_ARN", "DefaultAccountCreatedTopicArn")
	resetQueueURL = Config.GetEnvVar("RESET_SQS_URL", "DefaultResetSQSUrl")
	allowedRegions = strings.Split(Config.GetEnvVar("ALLOWED_REGIONS", "us-east-1"), ",")
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	CurrentAccountID = &req.RequestContext.AccountID
	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	Dao = newDBer()
	AWSSession = newAWSSession()
	Queue = common.SQSQueue{Client: sqs.New(AWSSession)}
	SnsSvc = &common.SNS{Client: sns.New(AWSSession)}
	TokenSvc = common.STS{Client: sts.New(AWSSession)}

	StorageSvc = common.S3{
		Client:  s3.New(AWSSession),
		Manager: s3manager.NewDownloader(AWSSession),
	}

	RoleManager = &rolemanager.IAMRoleManager{}
	// Send Lambda requests to the router
	lambda.Start(Handler)
}

func newDBer() db.DBer {
	dao, err := db.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize database: %s", err)
		log.Fatal(errorMessage)
	}

	return dao
}

func newAWSSession() *session.Session {
	awsSession, err := session.NewSession()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create AWS session: %s", err)
		log.Fatal(errorMessage)
	}
	return awsSession
}

// WriteAPIResponse - Writes the response out to the provided ResponseWriter
func WriteAPIResponse(w http.ResponseWriter, status int, body interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// ErrorHandler returns the appropriate HTTP code depending on the error recieved
func ErrorHandler(w http.ResponseWriter, err error) {
	var status int
	var code string
	var message string
	// Print the Error Message
	log.Printf("Error: %+v\n", err)

	// Determine status code
	if gErrors.Is(err, errors.ErrNotFound) {
		status = http.StatusNotFound
		code = "NotFound"
		message = err.Error()
	} else if gErrors.Is(err, &errors.ErrValidation{}) {
		status = http.StatusBadRequest
		code = "RequestValidationError"
		message = err.Error()
	} else if gErrors.Is(err, errors.ErrConflict) {
		status = http.StatusConflict
		code = "Conflict"
		message = err.Error()
	} else {
		status = http.StatusInternalServerError
		code = "ServerError"
		message = err.Error()
	}
	WriteAPIResponse(
		w,
		status,
		response.CreateErrorResponse(
			code,
			message,
		),
	)
}
