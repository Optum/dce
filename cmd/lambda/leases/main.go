package main

import (
	"context"
	"fmt"
	"net/url"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
	UsageStepFunctionArn     string  `env:"USAGE_STEP_FUNCTION_ARN" defaultEnv:"DefaultUsageStepFunctionArn"`
}

var (
	muxLambda *gorillamux.GorillaMuxAdapter
	//CurrentAccountID is the ID where the request is being created
	// Services handles the configuration of the AWS services
	Services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	Settings *leaseControllerConfiguration
)

var (
	baseRequest url.URL
	userDetailsMiddleware api.UserDetailsMiddleware
)

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
	userDetailsMiddleware = api.UserDetailsMiddleware{}
	r.Use(userDetailsMiddleware.Middleware)
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
		WithStepFunctions().
		WithLeaseService().
		WithAccountService().
		WithUserDetailer().
		Build()
	if err != nil {
		panic(err)
	}

	Services = svcBldr
}

// Handler - Handle the lambda function
func Handler(_ context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Provide configuration to middleware
	userDetailsMiddleware.UserDetailer = Services.UserDetailer()
	userDetailsMiddleware.GorillaMuxAdapter = muxLambda

	// Set baseRequest information lost by integration with gorilla mux
	baseRequest = url.URL{}
	baseRequest.Scheme = req.Headers["X-Forwarded-Proto"]
	baseRequest.Host = req.Headers["Host"]
	baseRequest.Path = fmt.Sprintf("%s%s", req.RequestContext.Stage, req.Path)

	return muxLambda.Proxy(req)
}

func main() {
	lambda.Start(Handler)
}
