package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

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
	// CurrentAccountID is the ID where the request is being created
	currentAccountID *string
	// Services handles the configuration of the AWS services
	services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	settings *accountControllerConfiguration
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
			GetAllAccounts,
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
	settings = &accountControllerConfiguration{}
	if err := cfgBldr.Unmarshal(settings); err != nil {
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
		Build()

	if err == nil {
		services = svcBldr
	}
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Set the Current Account from the request
	currentAccountID = &req.RequestContext.AccountID
	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Send Lambda requests to the router
	lambda.Start(Handler)
}
