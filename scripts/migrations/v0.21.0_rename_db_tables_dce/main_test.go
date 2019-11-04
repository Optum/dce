package main

import (
	"github.com/Optum/Redbox/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/require"
	"log"
	"strconv"
	"testing"
)

func TestMigrateV0210(t *testing.T) {
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	srcTableName := common.RequireEnv("TEST_MIGRATION_0210_SRC_TABLE_NAME")
	dstTableName := common.RequireEnv("TEST_MIGRATION_0210_DST_TABLE_NAME")
	_ = dstTableName

	// Truncate test tables before and after
	truncateAccountTable(t, dynDB, srcTableName)
	truncateAccountTable(t, dynDB, dstTableName)
	defer truncateAccountTable(t, dynDB, srcTableName)
	defer truncateAccountTable(t, dynDB, dstTableName)

	t.Run("Should migrate lots of records", func(t *testing.T) {
		var RECORDS_COUNT = 1000

		// Prepare a bunch of records
		var records []map[string]*dynamodb.AttributeValue
		for i := range make([]int, RECORDS_COUNT) {
			records = append(records, map[string]*dynamodb.AttributeValue{
				"AccountStatus": {
					S: aws.String("Leased"),
				},
				"CreatedOn": {
					N: aws.String("1569441730"),
				},
				"PrincipalPolicyHash": {
					S: aws.String("\"85XXXXX1c435a8c0e65490\""),
				},
				"PrincipalRoleArn": {
					S: aws.String("arn:aws:iam::123456789012:role/DCEPrincipal"),
				},
				"AdminRoleArn": {
					S: aws.String("arn:aws:iam::123456789012:role/AdminRole"),
				},
				"Id": {
					S: aws.String("account_" + strconv.Itoa(i)),
				},
				"LastModifiedOn": {
					N: aws.String("1572387715"),
				},
				"Metadata": {
					M: map[string]*dynamodb.AttributeValue{
						"foo": {S: aws.String("bar")},
					},
				},
			})
		}

		// Write the records to the source table
		for i, item := range records {
			_, err := dynDB.PutItem(&dynamodb.PutItemInput{
				TableName: &srcTableName,
				Item:      item,
			})
			require.Nil(t, err)
			log.Printf("Put fixture record %d/%d to %s", i+1, len(records), srcTableName)
		}

		// Run migration
		err := migrate(&migrateInput{
			db: dynDB,
			tables: map[string]string{
				srcTableName: dstTableName,
			},
		})
		require.Nil(t, err)

		// Dest table should have all the records from our source able
		dstScan, err := dynDB.Scan(&dynamodb.ScanInput{
			TableName:      &dstTableName,
			ConsistentRead: aws.Bool(true),
		})
		require.Nil(t, err)
		require.Len(t, dstScan.Items, RECORDS_COUNT)
		// Destination table should have all the records from our src table
		require.ElementsMatch(t, dstScan.Items, records)

		// Source table should still have all the same records
		// (should be non-destructive)
		srcScan, err := dynDB.Scan(&dynamodb.ScanInput{
			TableName:      &srcTableName,
			ConsistentRead: aws.Bool(true),
		})
		require.Nil(t, err)
		require.Len(t, srcScan.Items, RECORDS_COUNT)
		// Destination table should have all the records from our src table
		require.ElementsMatch(t, srcScan.Items, records)
	})

}

func truncateAccountTable(t *testing.T, dynDB *dynamodb.DynamoDB, accountTableName string) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the RedboxAccount table
	scanResult, err := dynDB.Scan(
		&dynamodb.ScanInput{
			TableName: aws.String(accountTableName),
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
	var batch []*dynamodb.WriteRequest
	for i, req := range deleteRequests {
		batch = append(batch, req)

		// Batch API only allows 25 at a time
		if i%25 == 0 || i == len(deleteRequests)-1 {
			_, err = dynDB.BatchWriteItem(
				&dynamodb.BatchWriteItemInput{
					RequestItems: map[string][]*dynamodb.WriteRequest{
						accountTableName: batch,
					},
				},
			)
			require.Nil(t, err)
			batch = []*dynamodb.WriteRequest{}
			log.Printf("Deleted records %d - %d / %d from %s", i-24, i+1, len(deleteRequests), accountTableName)
		}
	}
}
