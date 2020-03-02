package main

import (
	"context"
	"fmt"
	"net/url"

	"log"

	"github.com/Optum/dce/pkg/api"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/Optum/dce/pkg/config"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

type leaseControllerConfiguration struct {
	Debug string `env:"DEBUG" defaultEnv:"false"`
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
	//cognitoUserPoolId        string
	//cognitoAdminName         string
	userDetailsMiddleware api.UserDetailsMiddleware
)

func init() {
	initConfig()
	log.Println("Cold start; creating router for /usage")

	usageRoutes := api.Routes{
		api.Route{
			"ListUsageByPrincipal",
			"GET",
			"/usage/principal/{principalID}",
			api.EmptyQueryString,
			ListPrincipalUsageByPrincipal,
		},
		api.Route{
			"GetLeaseUsageSummaryByLease",
			"GET",
			"/usage/lease/{leaseID}/summary",
			api.EmptyQueryString,
			GetLeaseUsageSummaryByLease,
		},
		api.Route{
			"ListPrincipalUsage",
			"GET",
			"/usage/principal",
			api.EmptyQueryString,
			ListPrincipalUsage,
		},
		api.Route{
			"ListLeaseUsage",
			"GET",
			"/usage/lease",
			api.EmptyQueryString,
			ListLeaseUsageSummary,
		},
	}
	r := api.NewRouter(usageRoutes)
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
		WithUsageService().
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
