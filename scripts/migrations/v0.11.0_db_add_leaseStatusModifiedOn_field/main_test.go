package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

func TestMigrationV0_11_0(t *testing.T) {
	// Load Terraform outputs, to get DB table names
	tfOpts := &terraform.Options{
		TerraformDir: "../../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	leaseTable := tfOut["redbox_account_lease_db_table_name"].(string)

	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	// Clean DB table to start, and clean again when we're done
	truncateLeaseTable(t, dynDB, leaseTable)
	defer truncateLeaseTable(t, dynDB, leaseTable)

	// Add data to the assignments table
	assignments := []map[string]*dynamodb.AttributeValue{
		{
			"AccountId":      attrStr("Account1"),
			"PrincipalId":    attrStr("User1"),
			"LeaseStatus":    attrStr("Active"),
			"CreatedOn":      attrInt(1234567890),
			"LastModifiedOn": attrInt(1234567890),
		},
		{
			"AccountId":      attrStr("Account2"),
			"PrincipalId":    attrStr("User2"),
			"LeaseStatus":    attrStr("Active"),
			"CreatedOn":      attrInt(1234567890),
			"LastModifiedOn": attrInt(1234567890),
		},
	}
	for _, assign := range assignments {
		_, err := dynDB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(leaseTable),
			Item:      assign,
		})
		require.Nil(t, err)
	}

	// Run the Migration
	modifiedTime := time.Now().Unix()
	_, err := migrationV11(&migrationV11Input{
		leaseTableName: leaseTable,
		dynDB:          dynDB,
		leaseModTime:   modifiedTime,
	})
	require.Nil(t, err)

	// Scan the Leases table
	leaseScanRes, err := dynDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String(leaseTable),
	})
	require.Nil(t, err)
	require.Len(t, leaseScanRes.Items, 2)

	// Group Leases by AccountId, so we can easily look them up, and compare
	// with expectations
	leasesByAcctID := map[string]map[string]*dynamodb.AttributeValue{}
	for _, lease := range leaseScanRes.Items {
		leasesByAcctID[*lease["AccountId"].S] = lease
	}

	// Lease for Account1
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"AccountId":             attrStr("Account1"),
		"PrincipalId":           attrStr("User1"),
		"LeaseStatus":           attrStr("Active"),
		"CreatedOn":             attrInt(1234567890),
		"LastModifiedOn":        attrInt(modifiedTime),
		"LeaseStatusModifiedOn": attrInt(modifiedTime),
	}, leasesByAcctID["Account1"])
	// Lease for Account2
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"AccountId":             attrStr("Account2"),
		"PrincipalId":           attrStr("User2"),
		"LeaseStatus":           attrStr("Active"),
		"CreatedOn":             attrInt(1234567890),
		"LastModifiedOn":        attrInt(modifiedTime),
		"LeaseStatusModifiedOn": attrInt(modifiedTime),
	}, leasesByAcctID["Account2"])

}

func attrStr(str string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{
		S: aws.String(str),
	}
}
func attrInt(i int64) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{
		N: aws.String(strconv.FormatInt(i, 10)),
	}
}

func truncateLeaseTable(t *testing.T, dynDB *dynamodb.DynamoDB, leaseTableName string) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the RedboxAccount table
	scanResult, err := dynDB.Scan(
		&dynamodb.ScanInput{
			TableName: aws.String(leaseTableName),
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
	_, err = dynDB.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				leaseTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
}
