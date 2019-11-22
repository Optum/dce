package usage

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

/*
The `UsageDB` service abstracts all interactions
with the DynamoDB usage table
*/

// DB contains DynamoDB client and table names
type DB struct {
	// DynamoDB Client
	Client *dynamodb.DynamoDB
	// Name of the Usage table
	UsageTableName   string
	PartitionKeyName string
	SortKeyName      string
	// Use Consistend Reads when scanning or querying.  When possbile.
	ConsistendRead bool
}

// Usage item
type Usage struct {
	PrincipalID  string  `json:"PrincipalId"`  // User Principal ID
	AccountID    string  `json:"AccountId"`    // AWS Account ID
	StartDate    int64   `json:"StartDate"`    // Usage start date Epoch Timestamp
	EndDate      int64   `json:"EndDate"`      // Usage ends date Epoch Timestamp
	CostAmount   float64 `json:"CostAmount"`   // Cost Amount for given period
	CostCurrency string  `json:"CostCurrency"` // Cost currency
	TimeToLive   int64   `json:"TimeToLive"`   // ttl attribute
}

// The Service interface includes all methods used by the DB struct to interact with
// Usage DynamoDB. This is useful if we want to mock the DB service.
type Service interface {
	PutUsage(input Usage) error
	GetUsageByDateRange(startDate time.Time, endDate time.Time) ([]*Usage, error)
	GetUsageByPrincipal(startDate time.Time, principalID string) ([]*Usage, error)
}

// PutUsage adds an item to Usage DB
func (db *DB) PutUsage(input Usage) error {
	item, err := dynamodbattribute.MarshalMap(input)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to add usage record for start date \"%d\" and PrincipalID \"%s\": %s.", input.StartDate, input.PrincipalID, err)
		log.Print(errorMessage)
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

// GetUsageByDateRange returns usage amount for all leases for input date range
// startDate and endDate are epoch Unix dates
func (db *DB) GetUsageByDateRange(startDate time.Time, endDate time.Time) ([]*Usage, error) {

	scanOutput := make([]*dynamodb.QueryOutput, 0)

	// Convert startDate to the start time of that day
	usageStartDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	usageEndDate := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, time.UTC)

	if usageEndDate.Sub(usageStartDate) < 0 {
		errorMessage := fmt.Sprintf("UsageStartDate \"%d\" should be before usageEndDate \"%d\".", usageStartDate.Unix(), usageEndDate.Unix())
		log.Print(errorMessage)
		return nil, nil
	}

	for {

		var resp, err = db.Client.Query(getQueryInput(db.UsageTableName, usageStartDate, nil, db.ConsistendRead))
		if err != nil {
			errorMessage := fmt.Sprintf("Failed to query usage record for start date \"%s\": %s.", startDate, err)
			log.Print(errorMessage)
			return nil, err
		}
		scanOutput = append(scanOutput, resp)

		// pagination
		for len(resp.LastEvaluatedKey) > 0 {
			var resp, err = db.Client.Query(getQueryInput(db.UsageTableName, usageStartDate, resp.LastEvaluatedKey, db.ConsistendRead))
			if err != nil {
				errorMessage := fmt.Sprintf("Failed to query usage record for start date \"%s\": %s.", startDate, err)
				log.Print(errorMessage)
				return nil, err
			}
			scanOutput = append(scanOutput, resp)
		}

		// increment startdate by a day
		usageStartDate = usageStartDate.AddDate(0, 0, 1)

		// continue to get usage till usageEndDate
		if usageEndDate.Sub(usageStartDate) < 0 {
			break
		}
	}

	usageRecords := []*Usage{}
	for _, s := range scanOutput {

		// Create the array of Usage records
		for _, r := range s.Items {
			n, err := unmarshalUsageRecord(r)
			if err != nil {
				return nil, err
			}
			usageRecords = append(usageRecords, n)
		}
	}

	return usageRecords, nil
}

// GetUsageByPrincipal returns usage amount for all leases for input Principal
// startDate is epoch Unix date
func (db *DB) GetUsageByPrincipal(startDate time.Time, principalID string) ([]*Usage, error) {

	output := make([]*Usage, 0)

	// Convert startDate to the start time of that day
	usageStartDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	currentDate := time.Now()
	usageEndDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 0, time.UTC)

	if usageEndDate.Sub(usageStartDate) < 0 {
		errorMessage := fmt.Sprintf("UsageStartDate \"%d\" should be a past date or today's date \"%d\".", usageStartDate.Unix(), usageEndDate.Unix())
		log.Print(errorMessage)
		return nil, nil
	}

	for {

		var resp, err = db.Client.GetItem(getInputForGetUsageByPrincipalID(db, usageStartDate, principalID, db.ConsistendRead))
		if err != nil {
			errorMessage := fmt.Sprintf("Failed to query usage record for start date \"%s\": %s.", startDate, err)
			log.Print(errorMessage)
			return nil, err
		}

		item := Usage{}

		err = dynamodbattribute.UnmarshalMap(resp.Item, &item)
		if err != nil {
			errorMessage := fmt.Sprintf("Failed to unmarshal Record, %v", err)
			log.Print(errorMessage)
			return nil, err
		}

		output = append(output, &item)

		// increment startdate by a day
		usageStartDate = usageStartDate.AddDate(0, 0, 1)

		// continue to get usage till usageEndDate
		if usageEndDate.Sub(usageStartDate) < 0 {
			break
		}
	}

	return output, nil
}

// New creates a new usage DB Service struct,
// with all the necessary fields configured.
func New(client *dynamodb.DynamoDB, usageTableName string, partitionKeyName string, sortKeyName string) *DB {
	return &DB{
		Client:           client,
		UsageTableName:   usageTableName,
		PartitionKeyName: partitionKeyName,
		SortKeyName:      sortKeyName,
		ConsistendRead:   false,
	}
}

/*
NewFromEnv creates a DB instance configured from environment variables.
Requires env vars for:

- AWS_CURRENT_REGION
- USAGE_CACHE_DB
*/
func NewFromEnv() (*DB, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(common.RequireEnv("AWS_CURRENT_REGION")),
		),
		common.RequireEnv("USAGE_CACHE_DB"),
		"StartDate",
		"PrincipalId",
	), nil
}

func unmarshalUsageRecord(dbResult map[string]*dynamodb.AttributeValue) (*Usage, error) {
	usageRecord := Usage{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &usageRecord)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to unmarshal usage record \"%v\": %s.", dbResult, err)
		log.Print(errorMessage)
		return nil, err
	}

	return &usageRecord, nil
}

func getQueryInput(tableName string, startDate time.Time, startKey map[string]*dynamodb.AttributeValue, consistentRead bool) *dynamodb.QueryInput {

	return &dynamodb.QueryInput{
		TableName:         aws.String(tableName),
		ExclusiveStartKey: startKey,
		KeyConditions: map[string]*dynamodb.Condition{
			"StartDate": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						N: aws.String(strconv.FormatInt(startDate.Unix(), 10)),
					},
				},
			},
		},
		ConsistentRead: aws.Bool(consistentRead),
	}
}

// getInputForGetUsageByPrincipalID returns a GetItemInput for given inputs
func getInputForGetUsageByPrincipalID(d *DB, startDate time.Time, principalID string, consistentRead bool) *dynamodb.GetItemInput {
	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(d.UsageTableName),
		Key: map[string]*dynamodb.AttributeValue{
			d.PartitionKeyName: {
				N: aws.String(strconv.FormatInt(startDate.Unix(), 10)),
			},
			d.SortKeyName: {
				S: aws.String(principalID),
			},
		},
		ConsistentRead: aws.Bool(consistentRead),
	}
	return &getItemInput
}
