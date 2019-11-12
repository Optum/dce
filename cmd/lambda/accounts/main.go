package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	// AWSSession - The AWS session
	AWSSession *session.Session
	// RoleManager - Manages the roles
	RoleManager rolemanager.RoleManager
	// Dao - Database service
	Dao db.DBer
	// SnsSvc - SNS service
	SnsSvc common.Notificationer
	// ProvisionTopicArn - ARN for SNS topic for the provisioner.
	ProvisionTopicArn *string
	// Queue - SQS Queue client
	Queue common.Queue
	// TokenSvc - Token service client
	TokenSvc common.TokenService
	// StorageSvc - Storage service client
	StorageSvc common.Storager
	// Config - The configuration client
	Config common.DefaultEnvConfig
)

func init() {
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
		api.Route{
			"GetAllAccounts",
			"GET",
			"/accounts",
			api.EmptyQueryString,
			GetAllAccounts,
		}, // routes that use query strings for matchers.
		api.Route{
			"GetAccountByID",
			"GET",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			GetAccountByID,
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

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Setup services
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
