package testutils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Remove all records from the Account table
func TruncateAccountTable(t *testing.T, dbSvc dynamodbiface.DynamoDBAPI, tableName string) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(tableName),
			ConsistentRead: aws.Bool(true),
		},
	)
	require.Nil(t, err)

	if len(scanResult.Items) < 1 {
		return
	}

	// Populate a list of `DeleteRequests` for each item we found in the table
	var deleteRequests []*dynamodb.WriteRequest
	for _, item := range scanResult.Items {
		deleteRequests = append(deleteRequests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"Id": item["Id"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				tableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

/*
Remove all records from the Lease table
*/
func TruncateLeaseTable(t *testing.T, dbSvc dynamodbiface.DynamoDBAPI, tableName string) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(tableName),
			ConsistentRead: aws.Bool(true),
		},
	)
	require.Nil(t, err)

	if len(scanResult.Items) < 1 {
		return
	}

	// Populate a list of `DeleteRequests` for each
	// item we found in the table
	var deleteRequests []*dynamodb.WriteRequest
	for _, item := range scanResult.Items {
		deleteRequests = append(deleteRequests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"AccountId":   item["AccountId"],
					"PrincipalId": item["PrincipalId"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				tableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

func TruncatePrincipalTable(t *testing.T, dbSvc dynamodbiface.DynamoDBAPI, tableName string) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(tableName),
			ConsistentRead: aws.Bool(true),
		},
	)
	require.Nil(t, err)

	if len(scanResult.Items) < 1 {
		return
	}

	// Populate a list of `DeleteRequests` for each
	// item we found in the table
	var deleteRequests []*dynamodb.WriteRequest
	for _, item := range scanResult.Items {
		deleteRequests = append(deleteRequests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"SK":          item["SK"],
					"PrincipalId": item["PrincipalId"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				tableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

type usageLeaseData struct {
	PrincipalID  string  `json:"PrincipalId" dynamodbav:"PrincipalId"`
	LeaseID      string  `json:"LeaseId" dynamodbav:"LeaseId"`
	Date         int64   `json:"Date" dynamodbav:"Date"`
	CostAmount   float64 `json:"CostAmount" dynamodbav:"CostAmount"`
	CostCurrency string  `json:"CostCurrency" dynamodbav:"CostCurrency"`
	BudgetAmount float64 `json:"BudgetAmount" dynamodbav:"BudgetAmount"`
	SK           string  `json:"SK" dynamodbav:"SK"`
	TimeToLive   int64   `json:"TimeToLive" dynamodbav:"TimeToLive"`
}

func LoadUsageLeaseRecords(t *testing.T, dbSvc dynamodbiface.DynamoDBAPI, tableName string, files ...string) {

	for _, file := range files {
		f, err := os.Open(filepath.Clean(file))
		assert.Nil(t, err)

		byteValue, _ := ioutil.ReadAll(f)

		var records []usageLeaseData
		_ = json.Unmarshal(byteValue, &records)

		items := map[string][]*dynamodb.WriteRequest{}
		for _, record := range records {
			// cleanup of usage record dates
			now := time.Now().UTC()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			weeksBetween := int(math.Round(today.Sub(time.Unix(record.Date, 0).UTC()).Hours() / 24 / 7))

			newDate := time.Unix(record.Date, 0).UTC().AddDate(0, 0, weeksBetween*7).Unix()
			record.SK = strings.Replace(record.SK, fmt.Sprintf("%d", record.Date), fmt.Sprintf("%d", newDate), 1)
			record.Date = newDate

			putMap, _ := dynamodbattribute.MarshalMap(record)
			items[tableName] = append(items[tableName],
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: putMap,
					},
				},
			)
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: items,
		}

		_, err = dbSvc.BatchWriteItem(input)
		assert.Nil(t, err)
	}
}

type usagePrincipalData struct {
	PrincipalID  string  `json:"PrincipalId" dynamodbav:"PrincipalId"`
	Date         int64   `json:"Date" dynamodbav:"Date"`
	CostAmount   float64 `json:"CostAmount" dynamodbav:"CostAmount"`
	CostCurrency string  `json:"CostCurrency" dynamodbav:"CostCurrency"`
	SK           string  `json:"SK" dynamodbav:"SK"`
	TimeToLive   int64   `json:"TimeToLive" dynamodbav:"TimeToLive"`
}

func LoadUsagePrincipalRecords(t *testing.T, dbSvc dynamodbiface.DynamoDBAPI, tableName string, files ...string) {

	for _, file := range files {
		f, err := os.Open(filepath.Clean(file))
		assert.Nil(t, err)

		byteValue, _ := ioutil.ReadAll(f)

		var records []usagePrincipalData
		_ = json.Unmarshal(byteValue, &records)

		items := map[string][]*dynamodb.WriteRequest{}
		for _, record := range records {
			// cleanup of usage record dates
			now := time.Now().UTC()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			weeksBetween := int(math.Round(today.Sub(time.Unix(record.Date, 0).UTC()).Hours() / 24 / 7))

			newDate := time.Unix(record.Date, 0).UTC().AddDate(0, 0, weeksBetween*7).Unix()
			record.SK = strings.Replace(record.SK, fmt.Sprintf("%d", record.Date), fmt.Sprintf("%d", newDate), 1)
			record.Date = newDate

			putMap, _ := dynamodbattribute.MarshalMap(record)
			items[tableName] = append(items[tableName],
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: putMap,
					},
				},
			)
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: items,
		}

		_, err = dbSvc.BatchWriteItem(input)
		assert.Nil(t, err)
	}
}

// Lease is a type corresponding to a Lease
// table record
type leaseData struct {
	AccountID                string   `json:"AccountId" dynamodbav:"AccountId"`
	PrincipalID              string   `json:"PrincipalId" dynamodbav:"PrincipalId"`
	ID                       string   `json:"Id" dynamodbav:"Id"`
	Status                   string   `json:"LeaseStatus" dynamodbav:"LeaseStatus" `
	StatusReason             string   `json:"LeaseStatusReason" dynamodbav:"LeaseStatusReason"`
	CreatedOn                int64    `json:"CreatedOn" dynamodbav:"CreatedOn" `
	LastModifiedOn           int64    `json:"LastModifiedOn" dynamodbav:"LastModifiedOn" `
	BudgetAmount             float64  `json:"BudgetAmount" dynamodbav:"BudgetAmount"`
	BudgetCurrency           string   `json:"BudgetCurrency" dynamodbav:"BudgetCurrency"`
	BudgetNotificationEmails []string `json:"BudgetNotificationEmails" dynamodbav:"BudgetNotificationEmails"`
	StatusModifiedOn         int64    `json:"LeaseStatusModifiedOn" dynamodbav:"LeaseStatusModifiedOn" `
	ExpiresOn                int64    `json:"ExpiresOn" dynamodbav:"ExpiresOn" `
}

func LoadLeaseRecords(t *testing.T, dbSvc dynamodbiface.DynamoDBAPI, tableName string, files ...string) {

	for _, file := range files {
		f, err := os.Open(filepath.Clean(file))
		assert.Nil(t, err)

		byteValue, _ := ioutil.ReadAll(f)

		var records []leaseData
		_ = json.Unmarshal(byteValue, &records)

		items := map[string][]*dynamodb.WriteRequest{}
		for _, record := range records {
			// cleanup of usage record dates
			now := time.Now().UTC()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			weeksBetween := int(math.Round(today.Sub(time.Unix(record.CreatedOn, 0).UTC()).Hours() / 24 / 7))

			record.CreatedOn = time.Unix(record.CreatedOn, 0).UTC().AddDate(0, 0, weeksBetween*7).Unix()
			record.StatusModifiedOn = time.Unix(record.StatusModifiedOn, 0).UTC().AddDate(0, 0, weeksBetween*7).Unix()
			record.LastModifiedOn = time.Unix(record.StatusModifiedOn, 0).UTC().AddDate(0, 0, weeksBetween*7).Unix()
			record.ExpiresOn = time.Unix(record.StatusModifiedOn, 0).UTC().AddDate(0, 0, weeksBetween*7).Unix()

			putMap, _ := dynamodbattribute.MarshalMap(record)
			items[tableName] = append(items[tableName],
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: putMap,
					},
				},
			)
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: items,
		}

		_, err = dbSvc.BatchWriteItem(input)
		assert.Nil(t, err)
	}
}
