package tests

import (
	"testing"
	"time"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/require"
)

// Remove all records from the Account table
func truncateAccountTable(t *testing.T, dbSvc *db.DB) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(dbSvc.AccountTableName),
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
	_, err = dbSvc.Client.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dbSvc.AccountTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

/*
Remove all records from the Lease table
*/
func truncateLeaseTable(t *testing.T, dbSvc *db.DB) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(dbSvc.LeaseTableName),
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
	_, err = dbSvc.Client.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dbSvc.LeaseTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

func truncateDBTables(t *testing.T, dbSvc *db.DB) {
	truncateAccountTable(t, dbSvc)
	truncateLeaseTable(t, dbSvc)
}
