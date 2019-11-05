package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestMigrationV0_10_0(t *testing.T) {
	// Load Terraform outputs, to get DB table names
	tfOpts := &terraform.Options{
		TerraformDir: "../../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	accountTable := tfOut["redbox_account_db_table_name"].(string)
	leaseTable := tfOut["redbox_account_lease_db_table_name"].(string)
	assignmentTable := tfOut["redbox_assignment_db_table_name"].(string)

	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	// Clean DB table to start, and clean again when we're done
	truncateAllTables(t, dynDB, accountTable, assignmentTable, leaseTable)
	defer truncateAllTables(t, dynDB, accountTable, assignmentTable, leaseTable)

	// Add data to the Accounts table
	accounts := []map[string]*dynamodb.AttributeValue{
		{
			"Id":            attrStr("Account1"),
			"AccountStatus": attrStr("Ready"),
		},
		{
			"Id":            attrStr("Account2"),
			"AccountStatus": attrStr("Assigned"),
		},
		{
			"Id":            attrStr("Account3"),
			"AccountStatus": attrStr("NotReady"),
		},
		{
			"Id":            attrStr("Account4"),
			"AccountStatus": attrStr("Assigned"),
		},
	}
	for _, acct := range accounts {
		_, err := dynDB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(accountTable),
			Item:      acct,
		})
		require.Nil(t, err)
	}

	// Add data to the assignments table
	assignments := []map[string]*dynamodb.AttributeValue{
		{
			"AccountId":        attrStr("Account1"),
			"UserId":           attrStr("User1"),
			"AssignmentStatus": attrStr("Active"),
			"CreatedOn":        attrInt(1234567890),
			"LastModifiedOn":   attrInt(1234567890),
		},
		{
			"AccountId":        attrStr("Account2"),
			"UserId":           attrStr("User2"),
			"AssignmentStatus": attrStr("Active"),
			"CreatedOn":        attrInt(1234567890),
			"LastModifiedOn":   attrInt(1234567890),
		},
	}
	for _, assign := range assignments {
		_, err := dynDB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(assignmentTable),
			Item:      assign,
		})
		require.Nil(t, err)
	}

	// Run the Migration
	err := migrationV10(&migrationV10Input{
		assignmentTableName: assignmentTable,
		leaseTableName:      leaseTable,
		accountTableName:    accountTable,
		dynDB:               dynDB,
	})
	require.Nil(t, err)

	// Scan the Accounts Table
	accountScanRes, err := dynDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String(accountTable),
	})
	require.Nil(t, err)
	require.Len(t, accountScanRes.Items, 4)

	// Group Accounts by ID, so we can easily look them up, and compare
	// with expectations
	accountItemsById := map[string]map[string]*dynamodb.AttributeValue{}
	for _, acct := range accountScanRes.Items {
		accountItemsById[*acct["Id"].S] = acct
	}

	// Account 1: not changed
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"Id":            attrStr("Account1"),
		"AccountStatus": attrStr("Ready"),
	}, accountItemsById["Account1"])
	// Account 2: Changed status to Leased
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"Id":             attrStr("Account2"),
		"AccountStatus":  attrStr("Leased"),
		"LastModifiedOn": accountItemsById["Account2"]["LastModifiedOn"],
	}, accountItemsById["Account2"])
	// Account 3: Not changed
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"Id":            attrStr("Account3"),
		"AccountStatus": attrStr("NotReady"),
	}, accountItemsById["Account3"])
	// Account 4: Changed status to Leased
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"Id":             attrStr("Account4"),
		"AccountStatus":  attrStr("Leased"),
		"LastModifiedOn": accountItemsById["Account4"]["LastModifiedOn"],
	}, accountItemsById["Account4"])

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
		"AccountId":      attrStr("Account1"),
		"PrincipalId":    attrStr("User1"),
		"LeaseStatus":    attrStr("Active"),
		"CreatedOn":      attrInt(1234567890),
		"LastModifiedOn": leasesByAcctID["Account1"]["LastModifiedOn"],
	}, leasesByAcctID["Account1"])
	// Lease for Account2
	require.Equal(t, map[string]*dynamodb.AttributeValue{
		"AccountId":      attrStr("Account2"),
		"PrincipalId":    attrStr("User2"),
		"LeaseStatus":    attrStr("Active"),
		"CreatedOn":      attrInt(1234567890),
		"LastModifiedOn": leasesByAcctID["Account2"]["LastModifiedOn"],
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

func truncateAllTables(t *testing.T, dynDB *dynamodb.DynamoDB, accountTable string, assignmentTable string, leaseTable string) {
	truncateAccountTable(t, dynDB, accountTable)
	truncateAssignmentsTable(t, dynDB, assignmentTable)
	truncateLeaseTable(t, dynDB, leaseTable)
}

func truncateAccountTable(t *testing.T, dynDB *dynamodb.DynamoDB, accountTableName string) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
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
	_, err = dynDB.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				accountTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
}

func truncateLeaseTable(t *testing.T, dynDB *dynamodb.DynamoDB, leaseTableName string) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
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

func truncateAssignmentsTable(t *testing.T, dynDB *dynamodb.DynamoDB, assignmentsTableName string) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dynDB.Scan(
		&dynamodb.ScanInput{
			TableName: aws.String(assignmentsTableName),
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
					"AccountId": item["AccountId"],
					"UserId":    item["UserId"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dynDB.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				assignmentsTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
}
