package testutils

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
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
