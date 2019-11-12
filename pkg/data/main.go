package data

import (
	"fmt"
	"log"

	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

var (
	awsSession     *session.Session
	awsDynamoDB    dynamodbiface.DynamoDBAPI
	accountTable   string
	config         common.DefaultEnvConfig
	consistentRead bool
)

func init() {
	awsSession = newAWSSession()
	awsDynamoDB = newAWSDynamoClient()
	accountTable = config.GetEnvVar("ACCOUNT_DB", "Accounts")
	consistentRead = config.GetEnvBoolVar("CONSISTENT_READ", false)
}

func newAWSDynamoClient() dynamodbiface.DynamoDBAPI {
	return dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion(config.GetEnvVar("AWS_CURRENT_REGION", "us-east-1")),
	)
}

func newAWSSession() *session.Session {
	awsSession, err := session.NewSession()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create AWS session: %s", err)
		log.Fatal(errorMessage)
	}
	return awsSession
}
