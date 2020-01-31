package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/Optum/dce/pkg/config"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

type accountControllerConfiguration struct {
	Debug                       string   `env:"DEBUG" envDefault:"false"`
	PolicyName                  string   `env:"PRINCIPAL_POLICY_NAME" envDefault:"DCEPrincipalDefaultPolicy"`
	AccountCreatedTopicArn      string   `env:"ACCOUNT_CREATED_TOPIC_ARN" envDefault:"DefaultAccountCreatedTopicArn"`
	AccountDeletedTopicArn      string   `env:"ACCOUNT_DELETED_TOPIC_ARN"`
	ArtifactsBucket             string   `env:"ARTIFACTS_BUCKET" envDefault:"DefaultArtifactBucket"`
	PrincipalPolicyS3Key        string   `env:"PRINCIPAL_POLICY_S3_KEY" envDefault:"DefaultPrincipalPolicyS3Key"`
	PrincipalRoleName           string   `env:"PRINCIPAL_ROLE_NAME" envDefault:"DCEPrincipal"`
	PrincipalPolicyName         string   `env:"PRINCIPAL_POLICY_NAME"`
	PrincipalIAMDenyTags        []string `env:"PRINCIPAL_IAM_DENY_TAGS" envDefault:"DefaultPrincipalIamDenyTags"`
	PrincipalMaxSessionDuration int64    `env:"PRINCIPAL_MAX_SESSION_DURATION" envDefault:"100"`
	Tags                        []*iam.Tag
	ResetQueueURL               string   `env:"RESET_SQS_URL" envDefault:"DefaultResetSQSUrl"`
	AllowedRegions              []string `env:"ALLOWED_REGIONS" envDefault:"us-east-1"`
}

var (
	muxLambda *gorillamux.GorillaMuxAdapter
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
	RoleManager rolemanager.RoleManager
	baseRequest url.URL
	Config      common.DefaultEnvConfig
)

func init() {
	initConfig()

	log.Println("Cold start; creating router for /accounts")
	accountRoutes := api.Routes{
		// Routes with query strings always go first,
		// because the matcher will stop on the first match
		api.Route{
			"GetAccounts",
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
	r := api.NewRouter(accountRoutes)
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
	err := cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()
	if err != nil {
		log.Printf("Error: %+v", err)
	}
	svcBldr := &config.ServiceBuilder{Config: cfgBldr}

	_, err = svcBldr.
		// AWS services...
		WithDynamoDB().
		WithSTS().
		WithS3().
		WithSNS().
		WithSQS().
		// DCE services...
		WithStorageService().
		WithAccountDataService().
		WithEventService().
		WithAccountManagerService().
		WithAccountService().
		Build()
	if err != nil {
		panic(err)
	}

	Services = svcBldr

}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Set baseRequest information lost by integration with gorilla mux
	baseRequest = url.URL{}
	baseRequest.Scheme = req.Headers["X-Forwarded-Proto"]
	baseRequest.Host = req.Headers["Host"]
	baseRequest.Path = fmt.Sprintf("%s%s", req.RequestContext.Stage, req.Path)

	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	Dao = newDBer()
	AWSSession = newAWSSession()
	Queue = common.SQSQueue{Client: sqs.New(AWSSession)}
	SnsSvc = &common.SNS{Client: sns.New(AWSSession)}
	TokenSvc = common.STS{Client: sts.New(AWSSession)}

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
