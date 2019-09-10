package usage

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

/*
The `UsageDB` service abstracts all interactions
with the Redbox DynamoDB usage table
*/

// DB contains DynamoDB client and table names
type DB struct {
	// DynamoDB Client
	Client *dynamodb.DynamoDB
	// Name of the Usage table
	UsageTableName string
}

// Usage item
type Usage struct {
	PrincipalID  string
	AccountID    string
	StartDate    time.Time
	EndDate      time.Time
	CostAmount   string
	CostCurrency string
	BodyText     string
}

// The DBer interface includes all methods used by the DB struct to interact with
// Usage DynamoDB. This is useful if we want to mock the DB service.
type DBer interface {
	PutUsage(input Usage) error
	GetUsageByDaterange(startDate time.Time, endDate time.Time) ([]*Usage, error)
}

// PutUsage adds an item to Usage DB
func (db *DB) PutUsage(input Usage) error {
	item, err := dynamodbattribute.MarshalMap(input)
	if err != nil {
		return err
	}

	_, err = db.Client.PutItem(
		&dynamodb.PutItemInput{
			TableName: aws.String(db.UsageTableName),
			Item:      item,
		},
	)
	return err
}
