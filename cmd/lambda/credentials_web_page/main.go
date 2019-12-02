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

var muxLambda *gorillamux.GorillaMuxAdapter

type CredentialsWebPageConfig struct {
	AwsCurrentRegion     string `env:"AWS_CURRENT_REGION"`
	SitePathPrefix       string `env:"SITE_PATH_PREFIX"`
	ApigwDeploymentName  string `env:"APIGW_DEPLOYMENT_NAME"`
	IdentityPoolID       string `env:"PS_IDENTITY_POOL_ID"`
	UserPoolProviderName string `env:"PS_USER_POOL_PROVIDER_NAME"`
	UserPoolClientID     string `env:"PS_USER_POOL_CLIENT_ID"`
	UserPoolAppWebDomain string `env:"PS_USER_POOL_APP_WEB_DOMAIN"`
	UserPoolID           string `env:"PS_USER_POOL_ID"`
}

var Config *CredentialsWebPageConfig
var CfgBldr config.ConfigurationBuilder


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
	CfgBldr = &config.DefaultConfigurationBuilder{}
	CfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1")
	CfgBldr.WithEnv("SITE_PATH_PREFIX", "SITE_PATH_PREFIX", "sitePathPrefix")
	CfgBldr.WithEnv("APIGW_DEPLOYMENT_NAME", "APIGW_DEPLOYMENT_NAME", "apigwDeploymentName")
	CfgBldr.WithParameterStoreEnv("PS_IDENTITY_POOL_ID", "PS_IDENTITY_POOL_ID", "identityPoolID")
	CfgBldr.WithParameterStoreEnv("PS_USER_POOL_PROVIDER_NAME", "PS_USER_POOL_PROVIDER_NAME", "userPoolProviderName")
	CfgBldr.WithParameterStoreEnv("PS_USER_POOL_CLIENT_ID", "PS_USER_POOL_CLIENT_ID", "userPoolClientID")
	CfgBldr.WithParameterStoreEnv("PS_USER_POOL_APP_WEB_DOMAIN", "PS_USER_POOL_APP_WEB_DOMAIN", "userPoolAppWebDomain")
	CfgBldr.WithParameterStoreEnv("PS_USER_POOL_ID", "PS_USER_POOL_ID", "userPoolID")
	svcBldr := &config.DefaultAWSServiceBuilder{Config: CfgBldr}
	_, err := svcBldr.
		WithSSM().
		Build()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize parameter store: %s", err)
		log.Fatal(errorMessage)
	}

	Config = &CredentialsWebPageConfig{}
	if err := CfgBldr.Dump(Config); err != nil {
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
