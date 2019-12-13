package usage

import (
	"fmt"
	"log"
	"strconv"
	"strings"
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
	// Use Consistent Reads when scanning or querying.  When possbile.
	ConsistentRead bool
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

		var resp, err = db.Client.Query(getQueryInput(db.UsageTableName, usageStartDate, nil, db.ConsistentRead))
		if err != nil {
			errorMessage := fmt.Sprintf("Failed to query usage record for start date \"%s\": %s.", startDate, err)
			log.Print(errorMessage)
			return nil, err
		}
		scanOutput = append(scanOutput, resp)

		// pagination
		for len(resp.LastEvaluatedKey) > 0 {
			var resp, err = db.Client.Query(getQueryInput(db.UsageTableName, usageStartDate, resp.LastEvaluatedKey, db.ConsistentRead))
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

		var resp, err = db.Client.GetItem(getInputForGetUsageByPrincipalID(db, usageStartDate, principalID, db.ConsistentRead))
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

// GetUsageInput contains the filtering criteria for the GetUsage scan.
type GetUsageInput struct {
	StartKeys   map[string]string
	PrincipalID string
	AccountID   string
	StartDate   time.Time
	Limit       int64
}

// GetUsageOutput contains the scan results as well as the keys for retrieve the next page of the result set.
type GetUsageOutput struct {
	Results  []*Usage
	NextKeys map[string]string
}

// GetUsage takes a set of filtering criteria and scans the Usage table for the matching records.
func (db *DB) GetUsage(input GetUsageInput) (GetUsageOutput, error) {
	limit := int64(25)
	filters := make([]string, 0)
	filterValues := make(map[string]*dynamodb.AttributeValue)

	if input.Limit > 0 {
		limit = input.Limit
	}

	scanInput := &dynamodb.ScanInput{
		TableName:      aws.String(db.UsageTableName),
		Limit:          &limit,
		ConsistentRead: aws.Bool(db.ConsistentRead),
	}

	// Build the filter clauses.
	if input.StartDate != *new(time.Time) {
		filters = append(filters, "StartDate = :startDate")
		filterValues[":startDate"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(input.StartDate.Unix(), 10))}
	}

	if input.PrincipalID != "" {
		filters = append(filters, "PrincipalId = :principalId")
		filterValues[":principalId"] = &dynamodb.AttributeValue{S: aws.String(input.PrincipalID)}
	}

	if input.AccountID != "" {
		filters = append(filters, "AccountId = :accountId")
		filterValues[":accountId"] = &dynamodb.AttributeValue{S: aws.String(input.AccountID)}
	}

	if len(filters) > 0 {
		filterStatement := strings.Join(filters, " and ")
		scanInput.FilterExpression = &filterStatement
		scanInput.ExpressionAttributeValues = filterValues
	}

	if input.StartKeys != nil && len(input.StartKeys) > 0 {
		scanInput.ExclusiveStartKey = make(map[string]*dynamodb.AttributeValue)
		for k, v := range input.StartKeys {
			scanInput.ExclusiveStartKey[k] = &dynamodb.AttributeValue{S: aws.String(v)}
		}
	}

	output, err := db.Client.Scan(scanInput)

	// Parse the results and build the next keys if necessary.
	if err != nil {
		return GetUsageOutput{}, err
	}

	results := make([]*Usage, 0)

	for _, o := range output.Items {
		lease, err := unmarshalUsageRecord(o)
		if err != nil {
			return GetUsageOutput{}, err
		}
		results = append(results, lease)
	}

	nextKey := make(map[string]string)

	for k, v := range output.LastEvaluatedKey {
		nextKey[k] = *v.S
	}

	return GetUsageOutput{
		Results:  results,
		NextKeys: nextKey,
	}, nil
}

// New creates a new usage DB Service struct,
// with all the necessary fields configured.
func New(client *dynamodb.DynamoDB, usageTableName string, partitionKeyName string, sortKeyName string) *DB {
	return &DB{
		Client:           client,
		UsageTableName:   usageTableName,
		PartitionKeyName: partitionKeyName,
		SortKeyName:      sortKeyName,
		ConsistentRead:   false,
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
