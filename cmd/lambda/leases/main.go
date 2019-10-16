package main

import (
	"fmt"

	"log"

	"github.com/Optum/Redbox/pkg/api"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"
	"github.com/aws/aws-sdk-go/aws/session"
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
// to an SNS Topic for Provision and Decommission
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
	prov := &provision.AccountProvision{
		DBSvc: dao,
	}

	sqsClient := sqs.New(awsSession)
	queue := common.SQSQueue{
		Client: sqsClient,
	}

	provisionLeaseTopicARN := common.RequireEnv("PROVISION_TOPIC")
	accountDeletedTopicArn := common.RequireEnv("DECOMMISSION_TOPIC")
	resetQueueURL := common.RequireEnv("RESET_SQS_URL")

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
			Dao:           dao,
			Provisioner:   prov,
			SNS:           snsSvc,
			LeaseTopicARN: &provisionLeaseTopicARN,
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
