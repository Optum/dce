package usage

import (
	"strconv"
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
	PrincipalID  string  `json:"PrincipalId"`  // User Principal ID
	AccountID    string  `json:"AccountId"`    // AWS Account ID
	StartDate    int     `json:"StartDate"`    // Usage start date Epoch Timestamp
	EndDate      int     `json:"EndDate"`      // Usage ends date Epoch Timestamp
	CostAmount   float64 `json:"CostAmount"`   // Cost Amount for given period
	CostCurrency string  `json:"CostCurrency"` // Cost currency
	TimeToExist  int     `json:"TimeToExist"`  // ttl attribute
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

// GetUsageByDaterange returns usage amount for all leases
func (db *DB) GetUsageByDaterange(startDate int, days int) ([]*Usage, error) {

	scanOutput := make([]*dynamodb.QueryOutput, days)

	for i := 1; i <= days; i++ {

		var queryInput = &dynamodb.QueryInput{
			TableName: aws.String(db.UsageTableName),
			KeyConditions: map[string]*dynamodb.Condition{
				"StartDate": {
					ComparisonOperator: aws.String("EQ"),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							N: aws.String(strconv.Itoa(startDate)),
						},
					},
				},
			},
		}

		var resp, err = db.Client.Query(queryInput)
		if err != nil {
			return nil, err
		}
		scanOutput = append(scanOutput, resp)

		// pagination
		for len(resp.LastEvaluatedKey) > 0 {
			queryInput = &dynamodb.QueryInput{
				TableName:         aws.String(db.UsageTableName),
				ExclusiveStartKey: resp.LastEvaluatedKey,
				KeyConditions: map[string]*dynamodb.Condition{
					"StartDate": {
						ComparisonOperator: aws.String("EQ"),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								N: aws.String(strconv.Itoa(startDate)),
							},
						},
					},
				},
			}

			var resp, err = db.Client.Query(queryInput)
			if err != nil {
				return nil, err
			}

			scanOutput = append(scanOutput, resp)
		}

		startDate = startDate + 86400
	}

	usages := []*Usage{}
	for i := 1; i <= days; i++ {
		// Create the array of Usage records
		for _, r := range scanOutput[i].Items {
			n, err := unmarshalUsageRecord(r)
			if err != nil {
				return nil, err
			}
			usages = append(usages, n)
		}
	}

	return usages, nil
}

// New creates a new usage DB Service struct,
// with all the necessary fields configured.
func New(client *dynamodb.DynamoDB, usageTableName string) *DB {
	return &DB{
		Client:         client,
		UsageTableName: usageTableName,
	}
}

func unmarshalUsageRecord(dbResult map[string]*dynamodb.AttributeValue) (*Usage, error) {
	usageRecord := Usage{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &usageRecord)

	if err != nil {
		return nil, err
	}

	return &usageRecord, nil
}
