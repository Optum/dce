package main

import (
	"context"
	"fmt"
	"github.com/Optum/Redbox/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
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
	ListController   api.Controller
	DeleteController api.Controller
	GetController    api.Controller
	CreateController api.Controller
}

func (router *Router) route(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var res events.APIGatewayProxyResponse
	var err error
	switch {
	case req.HTTPMethod == http.MethodGet && strings.Compare(req.Path, "/accounts") == 0:
		res, err = router.ListController.Call(ctx, req)
	case req.HTTPMethod == http.MethodGet && strings.Contains(req.Path, "/accounts/"):
		res, err = router.GetController.Call(ctx, req)
	case req.HTTPMethod == http.MethodDelete && strings.Contains(req.Path, "/accounts/"):
		res, err = router.DeleteController.Call(ctx, req)
	case req.HTTPMethod == http.MethodPost && strings.Compare(req.Path, "/accounts") == 0:
		res, err = router.CreateController.Call(ctx, req)
	default:
		return response.NotFoundError(), nil
	}

	// Handle errors returned by controllers
	if err != nil {
		log.Printf("Controller error: %s", err)
		return response.ServerError(), nil
	}

	return res, nil
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
		ListController: listController{Dao: dao},
		GetController:  getController{Dao: dao},
		DeleteController: deleteController{
			Dao:                    dao,
			Queue:                  queue,
			ResetQueueURL:          common.RequireEnv("RESET_SQS_URL"),
			SNS:                    snsSvc,
			AccountDeletedTopicArn: common.RequireEnv("ACCOUNT_DELETED_TOPIC_ARN"),
			TokenService:           tokenSvc,
			AWSSession:             *awsSession,
			RoleManager:            &rolemanager.IAMRoleManager{},
			PrincipalRoleName:      common.RequireEnv("PRINCIPAL_ROLE_NAME"),
			PrincipalPolicyName:    common.RequireEnv("PRINCIPAL_POLICY_NAME"),
		},
		CreateController: createController{
			Dao:                         dao,
			Queue:                       queue,
			ResetQueueURL:               common.RequireEnv("RESET_SQS_URL"),
			SNS:                         snsSvc,
			AccountCreatedTopicArn:      common.RequireEnv("ACCOUNT_CREATED_TOPIC_ARN"),
			AWSSession:                  *awsSession,
			TokenService:                tokenSvc,
			RoleManager:                 &rolemanager.IAMRoleManager{},
			PrincipalRoleName:           common.RequireEnv("PRINCIPAL_ROLE_NAME"),
			PrincipalPolicyName:         common.RequireEnv("PRINCIPAL_POLICY_NAME"),
			PrincipalIAMDenyTags:        strings.Split(common.RequireEnv("PRINCIPAL_IAM_DENY_TAGS"), ","),
			PrincipalMaxSessionDuration: int64(common.RequireEnvInt("PRINCIPAL_MAX_SESSION_DURATION")),
			Tags: []*iam.Tag{
				{Key: aws.String("Terraform"), Value: aws.String("False")},
				{Key: aws.String("Source"), Value: aws.String("github.com/Optum/Redbox//cmd/lambda/accounts")},
				{Key: aws.String("Environment"), Value: aws.String(common.RequireEnv("TAG_ENVIRONMENT"))},
				{Key: aws.String("Contact"), Value: aws.String(common.RequireEnv("TAG_CONTACT"))},
				{Key: aws.String("AppName"), Value: aws.String(common.RequireEnv("TAG_APP_NAME"))},
			},
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
