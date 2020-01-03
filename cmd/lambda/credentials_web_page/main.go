package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

var (
	muxLambda *gorillamux.GorillaMuxAdapter
	// Services handles the configuration of the AWS services
	Services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	Settings *credentialsWebPageConfig
)

type credentialsWebPageConfig struct {
	Debug                string `env:"DEBUG" defaultEnv:"false"`
	AwsCurrentRegion     string `env:"AWS_CURRENT_REGION" defaultEnv:"us-east-1"`
	SitePathPrefix       string `env:"SITE_PATH_PREFIX" defaultEnv:"sitePathPrefix`
	ApigwDeploymentName  string `env:"APIGW_DEPLOYMENT_NAME" defaultEnv:"apigwDeploymentName"`
	IdentityPoolID       string `env:"PS_IDENTITY_POOL_ID" defaultEnv:"identityPoolID"`
	UserPoolProviderName string `env:"PS_USER_POOL_PROVIDER_NAME" defaultEnv:"userPoolProviderName"`
	UserPoolClientID     string `env:"PS_USER_POOL_CLIENT_ID" defaultEnv:"userPoolClientID"`
	UserPoolAppWebDomain string `env:"PS_USER_POOL_APP_WEB_DOMAIN" defaultEnv:"userPoolAppWebDomain"`
	UserPoolID           string `env:"PS_USER_POOL_ID" defaultEnv:"userPoolID"`
}

func init() {
	initConfig()

	log.Println("Cold start; creating router for /auth")
	authRoutes := api.Routes{
		api.Route{
			Name:        "GetAuthPage",
			Method:      "GET",
			Pattern:     "/auth",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetAuthPage,
		},
		api.Route{
			Name:        "GetAuthPageAssets",
			Method:      "GET",
			Pattern:     "/auth/public/{file}",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetAuthPageAssets,
		},
	}
	r := api.NewRouter(Services.Config, authRoutes)
	muxLambda = gorillamux.New(r)
}

func initConfig() {
	cfgBldr := &config.ConfigurationBuilder{}
	Settings = &credentialsWebPageConfig{}
	if err := cfgBldr.Unmarshal(Settings); err != nil {
		log.Fatalf("Could not load configuration: %s", err.Error())
	}

	svcBldr := &config.ServiceBuilder{Config: cfgBldr}
	_, err := svcBldr.
		WithSSM().
		Build()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize parameter store: %s", err)
		log.Fatal(errorMessage)
	}

	if err == nil {
		Services = svcBldr
	}
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Send Lambda requests to the router
	lambda.Start(Handler)
}
