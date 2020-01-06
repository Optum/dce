package main

import (
	"context"
	"fmt"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"log"
)

type credentialsWebPageConfig struct {
	AwsCurrentRegion     string `env:"AWS_CURRENT_REGION" defaultEnv:"us-east-1"`
	SitePathPrefix       string `env:"SITE_PATH_PREFIX" defaultEnv:"sitePathPrefix"`
	ApigwDeploymentName  string `env:"APIGW_DEPLOYMENT_NAME" defaultEnv:"apigwDeploymentName"`
	IdentityPoolID       string
	UserPoolProviderName string
	UserPoolClientID     string
	UserPoolAppWebDomain string
	UserPoolID           string
}

var (
	muxLambda *gorillamux.GorillaMuxAdapter
	// Settings - the configuration settings for the controller
	Settings *credentialsWebPageConfig
)

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
	r := api.NewRouter(authRoutes)
	muxLambda = gorillamux.New(r)
}

func initConfig() {
	cfgBldr := &config.ConfigurationBuilder{}
	Settings = &credentialsWebPageConfig{}
	if err := cfgBldr.Unmarshal(Settings); err != nil {
		log.Fatalf("Could not load configuration: %s", err.Error())
	}

	// load up the values into the various settings...
	_ = cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()
	cfgBldr.WithParameterStoreEnv("PS_IDENTITY_POOL_ID", "PS_IDENTITY_POOL_ID", "identityPoolID")
	cfgBldr.WithParameterStoreEnv("PS_USER_POOL_PROVIDER_NAME", "PS_USER_POOL_PROVIDER_NAME", "userPoolProviderName")
	cfgBldr.WithParameterStoreEnv("PS_USER_POOL_CLIENT_ID", "PS_USER_POOL_CLIENT_ID", "userPoolClientID")
	cfgBldr.WithParameterStoreEnv("PS_USER_POOL_APP_WEB_DOMAIN", "PS_USER_POOL_APP_WEB_DOMAIN", "userPoolAppWebDomain")
	cfgBldr.WithParameterStoreEnv("PS_USER_POOL_ID", "PS_USER_POOL_ID", "userPoolID")
	svcBldr := &config.ServiceBuilder{Config: cfgBldr}

	_, err := svcBldr.
		WithSSM().
		Build()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize parameter store: %s", err)
		log.Fatal(errorMessage)
	}

	if err := cfgBldr.Dump(Settings); err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize parameter store: %s", err)
		log.Fatal(errorMessage)
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
