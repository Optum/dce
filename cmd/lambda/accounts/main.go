package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/Optum/Redbox/pkg/api"
	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Router structure holds AccountController instance for request
type Router struct {
	GetAccountsController    api.Controller
	DeleteAccountController  api.Controller
	GetAccountByIDController api.Controller
	CreateAccountController  api.Controller
}

func (router *Router) route(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch {
	case req.HTTPMethod == http.MethodGet && strings.Compare(req.Path, "/accounts") == 0:
		return router.GetAccountsController.Call(ctx, req)
	case req.HTTPMethod == http.MethodGet && strings.Contains(req.Path, "/accounts/"):
		return router.GetAccountByIDController.Call(ctx, req)
	case req.HTTPMethod == http.MethodDelete && strings.Contains(req.Path, "/accounts/"):
		return router.DeleteAccountController.Call(ctx, req)
	case req.HTTPMethod == http.MethodPost && strings.Compare(req.Path, "/accounts") == 0:
		return router.CreateAccountController.Call(ctx, req)
	default:
		return response.CreateAPIErrorResponse(http.StatusNotFound, response.CreateErrorResponse(
			"NotFound", "Not found")), nil
	}
}

func main() {
	// Setup services
	dao := newDBer()
	awsSession := newAWSSession()
	queue := common.SQSQueue{Client: sqs.New(awsSession)}
	snsSvc := &common.SNS{Client: sns.New(awsSession)}
	tokenSvc := common.STS{Client: sts.New(awsSession)}

	// Configure the Router + Controllers
	router := &Router{
		GetAccountsController:    getAccountsController{Dao: dao},
		GetAccountByIDController: getAccountByIDController{Dao: dao},
		DeleteAccountController:  deleteAccountController{Dao: dao},
		CreateAccountController: createAccountController{
			Dao:                    dao,
			Queue:                  queue,
			ResetQueueURL:          common.RequireEnv("RESET_SQS_URL"),
			SNS:                    snsSvc,
			AccountCreatedTopicArn: common.RequireEnv("ACCOUNT_CREATED_TOPIC_ARN"),
			AWSSession:             *awsSession,
			TokenService:           tokenSvc,
		},
	}

	// Send Lambda requests to the router
	lambda.Start(router.route)
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
