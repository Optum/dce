package main

import (
	"context"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	muxLambda *gorillamux.GorillaMuxAdapter
	Services  *config.ServiceBuilder
)

func initConfig() {
	// Define required env vars
	cfgBldr := &config.ConfigurationBuilder{}
	_ = cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()

	// Define services we're going to need access to later
	Services = &config.ServiceBuilder{Config: cfgBldr}
	_, err := Services.
		WithLeaseService().
		WithUserDetailer().
		WithAccountManagerService().
		WithAccountService().
		Build()
	if err != nil {
		panic(err)
	}
}

func main() {
	initConfig()
	log.Println("Cold start; creating router for /leases/auth")

	routes := api.Routes{
		api.Route{
			"LeaseAuth",
			"POST",
			"/leases/{leaseID}/auth",
			api.EmptyQueryString,
			LeaseAuthHandler,
		},
		api.Route{
			"LeaseAuth",
			"POST",
			"/leases/auth",
			api.EmptyQueryString,
			LeaseAuthHandler,
		},
	}
	r := api.NewRouter(routes)
	muxLambda = gorillamux.New(r)

	lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return muxLambda.Proxy(req)
	})
}
