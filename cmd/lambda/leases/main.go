package main

import (
	"context"
	"fmt"
	"net/url"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/Optum/dce/pkg/config"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

type leaseControllerConfiguration struct {
	Debug                    string  `env:"DEBUG" defaultEnv:"false"`
	LeaseAddedTopicARN       string  `env:"LEASE_ADDED_TOPIC" defaultEnv:"DCEDefaultProvisionTopic"`
	DecommissionTopicARN     string  `env:"DECOMMISSION_TOPIC" defaultEnv:"DefaultDecommissionTopicArn"`
	CognitoUserPoolID        string  `env:"COGNITO_USER_POOL_ID" defaultEnv:"DefaultCognitoUserPoolId"`
	CognitoAdminName         string  `env:"COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME" defaultEnv:"DefaultCognitoAdminName"`
	PrincipalBudgetAmount    float64 `env:"PRINCIPAL_BUDGET_AMOUNT" defaultEnv:"1000.00"`
	PrincipalBudgetPeriod    string  `env:"PRINCIPAL_BUDGET_PERIOD" defaultEnv:"Weekly"`
	MaxLeaseBudgetAmount     float64 `env:"MAX_LEASE_BUDGET_AMOUNT" defaultEnv:"1000.00"`
	MaxLeasePeriod           int64   `env:"MAX_LEASE_PERIOD" defaultEnv:"704800"`
	DefaultLeaseLengthInDays int     `env:"DEFAULT_LEASE_LENGTH_IN_DAYS" defaultEnv:"7"`
}

const (
	Weekly = "WEEKLY"
)

var (
	muxLambda *gorillamux.GorillaMuxAdapter
	//CurrentAccountID is the ID where the request is being created
	// Services handles the configuration of the AWS services
	Services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	Settings *leaseControllerConfiguration
)

var (
	// Soon to be deprecated - Legacy support
	Config             common.DefaultEnvConfig
	awsSession         *session.Session
	dao                db.DBer
	snsSvc             common.Notificationer
	usageSvc           usage.Service
	leaseAddedTopicARN string
	//decommissionTopicARN     string
	principalBudgetAmount    float64
	principalBudgetPeriod    string
	maxLeaseBudgetAmount     float64
	maxLeasePeriod           int64
	defaultLeaseLengthInDays int
	baseRequest              url.URL
	//cognitoUserPoolId        string
	//cognitoAdminName         string
)

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for lease creation/destruction
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

func init() {
	initConfig()
	log.Println("Cold start; creating router for /leases")

	leasesRoutes := api.Routes{
		api.Route{
			"GetLeases",
			"GET",
			"/leases",
			api.EmptyQueryString,
			GetLeases,
		},
		api.Route{
			"GetLeaseByID",
			"GET",
			"/leases/{leaseID}",
			api.EmptyQueryString,
			GetLeaseByID,
		},
		api.Route{
			"DeleteLeaseByID",
			"DELETE",
			"/leases/{leaseID}",
			api.EmptyQueryString,
			DeleteLeaseByID,
		},
		api.Route{
			"DeleteLease",
			"DELETE",
			"/leases",
			api.EmptyQueryString,
			DeleteLease,
		},
		api.Route{
			"CreateLease",
			"POST",
			"/leases",
			api.EmptyQueryString,
			CreateLease,
		},
	}
	r := api.NewRouter(leasesRoutes)
	muxLambda = gorillamux.New(r)
}

// initConfig configures package-level variables
// loaded from env vars.
func initConfig() {
	cfgBldr := &config.ConfigurationBuilder{}
	Settings = &leaseControllerConfiguration{}
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
		WithLeaseDataService().
		WithLeaseService().
		Build()
	if err != nil {
		panic(err)
	}

	Services = svcBldr

	leaseAddedTopicARN = Config.GetEnvVar("LEASE_ADDED_TOPIC", "DCEDefaultProvisionTopic")
	//decommissionTopicARN = Config.GetEnvVar("DECOMMISSION_TOPIC", "DefaultDecommissionTopicArn")
	//cognitoUserPoolId = Config.GetEnvVar("COGNITO_USER_POOL_ID", "DefaultCognitoUserPoolId")
	//cognitoAdminName = Config.GetEnvVar("COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME", "DefaultCognitoAdminName")
	principalBudgetAmount = Config.GetEnvFloatVar("PRINCIPAL_BUDGET_AMOUNT", 1000.00)
	principalBudgetPeriod = Config.GetEnvVar("PRINCIPAL_BUDGET_PERIOD", Weekly)
	maxLeaseBudgetAmount = Config.GetEnvFloatVar("MAX_LEASE_BUDGET_AMOUNT", 1000.00)
	maxLeasePeriod = int64(Config.GetEnvIntVar("MAX_LEASE_PERIOD", 704800))
	defaultLeaseLengthInDays = Config.GetEnvIntVar("DEFAULT_LEASE_LENGTH_IN_DAYS", 7)
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	// requestUser := userDetails.GetUser(&req)
	// ctxWithUser := context.WithValue(ctx, api.DceCtxKey, *requestUser)
	// return muxLambda.ProxyWithContext(ctxWithUser, req)

	// Set baseRequest information lost by integration with gorilla mux
	baseRequest = url.URL{}
	baseRequest.Scheme = req.Headers["X-Forwarded-Proto"]
	baseRequest.Host = req.Headers["Host"]
	baseRequest.Path = fmt.Sprintf("%s%s", req.RequestContext.Stage, req.Path)

	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {

	awsSession = newAWSSession()
	// Create the Database Service from the environment
	dao = newDBer()
	snsSvc = &common.SNS{Client: sns.New(awsSession)}

	usageService, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	usageSvc = usageService

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
