package main

import (
	"fmt"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	PrincipalIDParam     = "principalId"
	AccountIDParam       = "accountId"
	NextPrincipalIDParam = "nextPrincipalId"
	NextAccountIDParam   = "nextAccountId"
	StatusParam          = "status"
	LimitParam           = "limit"
)

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for lease creation/destruction
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

// buildBaseURL returns a base API url from the request properties.
func buildBaseURL(req *events.APIGatewayProxyRequest) string {
	return fmt.Sprintf("https://%s/%s", req.Headers["Host"], req.RequestContext.Stage)
}

func main() {

	// Create the Database Service from the environment
	dao := newDBer()

	// Create the SNS Service
	awsSession := newAWSSession()
	snsSvc := &common.SNS{Client: sns.New(awsSession)}

	sqsClient := sqs.New(awsSession)
	queue := common.SQSQueue{
		Client: sqsClient,
	}

	usageSvc, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	leaseAddedTopicArn := common.RequireEnv("LEASE_ADDED_TOPIC")
	accountDeletedTopicArn := common.RequireEnv("DECOMMISSION_TOPIC")
	resetQueueURL := common.RequireEnv("RESET_SQS_URL")

	principalBudgetAmount := common.RequireEnvFloat("PRINCIPAL_BUDGET_AMOUNT")
	principalBudgetPeriod := common.RequireEnv("PRINCIPAL_BUDGET_PERIOD")
	maxLeaseBudgetAmount := common.RequireEnvFloat("MAX_LEASE_BUDGET_AMOUNT")
	maxLeasePeriod := common.RequireEnvInt("MAX_LEASE_PERIOD")

	router := &api.Router{
		ResourceName: "/leases",
		GetController: GetController{
			Dao: dao,
		},
		ListController: ListController{
			Dao: dao,
		},
		DeleteController: DeleteController{
			Dao:                    dao,
			SNS:                    snsSvc,
			AccountDeletedTopicArn: accountDeletedTopicArn,
			ResetQueueURL:          resetQueueURL,
			Queue:                  queue,
		},
		CreateController: CreateController{
			Dao:                      dao,
			SNS:                      snsSvc,
			LeaseAddedTopicARN:       &leaseAddedTopicArn,
			UsageSvc:                 usageSvc,
			PrincipalBudgetAmount:    &principalBudgetAmount,
			PrincipalBudgetPeriod:    &principalBudgetPeriod,
			MaxLeaseBudgetAmount:     &maxLeaseBudgetAmount,
			MaxLeasePeriod:           &maxLeasePeriod,
			DefaultLeaseLengthInDays: common.GetEnvInt("DEFAULT_LEASE_LENGTH_IN_DAYS", 7),
		},
		UserDetails: &api.UserDetails{
			CognitoUserPoolID:        common.RequireEnv("COGNITO_USER_POOL_ID"),
			RolesAttributesAdminName: common.RequireEnv("COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME"),
			CognitoClient:            cognitoidentityprovider.New(awsSession),
		},
	}

	lambda.Start(router.Route)
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
