package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
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

var muxLambda *gorillamux.GorillaMuxAdapter

// DbSvc - Create a DB service
var (
	DbSvc             db.DBer
	Provisioner       provision.Provisioner
	SnsSvc            common.Notificationer
	ProvisionTopicArn *string
)

func init() {

	log.Println("Gorilla Mux cold start")
	r := NewRouter()

	muxLambda = gorillamux.New(r)
}

// buildBaseURL returns a base API url from the request properties.
func buildBaseURL(req *events.APIGatewayProxyRequest) string {
	return fmt.Sprintf("https://%s/%s", req.Headers["Host"], req.RequestContext.Stage)
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {

	awsSession, _ := session.NewSession()
	ProvisionTopicArn = aws.String(os.Getenv("PROVISION_TOPIC"))
	SnsSvc = &common.SNS{Client: sns.New(awsSession)}
	DbSvc = newDBer()
	Provisioner = &provision.AccountProvision{
		DBSvc: DbSvc,
	}
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
