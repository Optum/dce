// v0.0.1_db_rename_assignment_to_lease.go
// This script is intended for one time use in the prod deployment of Redbox
// Its sole purpose is to copy existing records from RedboxAccountAssignmentProd
// to RedboxLeasetProd table.
//
// It is intended to be run as a Golang script:
// "go run v0.0.1_db_rename_assignment_to_lease.go"
//
// This script requires 3 environment variables to be set for its use:
// "export AWS_CURRENT_REGION=us-east-1"  - The region the database resides in
// "export SOURCE_DB=RedboxAccountAssignmentProd"  - Name of the Assignment table for Accounts
// "export LEASE_DB=RedboxLeasetProd"  - Name of the Lease table for Accounts. This is the new table to which items from SOURCE_DB will be added to

package main

import (
	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"strconv"
	"time"

	"fmt"
)

// AccountAssignment record
type AccountAssignment struct {
	AccountID        string `json:"AccountId"`
	UserID           string `json:"UserId"`
	AssignmentStatus string
	CreatedOn        int64
	LastModifiedOn   int64
}

// Lease record
type Lease struct {
	AccountID      string `json:"AccountId"`
	PrincipalID    string `json:"PrincipalId"`
	LeaseStatus    string
	CreatedOn      int64
	LastModifiedOn int64
}

// main is triggered
func main() {

	// Create DynamoDB Client
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	err := migrationV10(&migrationV10Input{
		assignmentTableName: common.RequireEnv("ASSIGNMENT_TABLE"),
		leaseTableName:      common.RequireEnv("LEASE_TABLE"),
		accountTableName:    common.RequireEnv("ACCOUNT_TABLE"),
		dynDB:               dynDB,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type migrationV10Input struct {
	assignmentTableName string
	leaseTableName      string
	accountTableName    string
	dynDB               *dynamodb.DynamoDB
}

// migrationV10 runs main logic
func migrationV10(input *migrationV10Input) error {

	/**
	 * Migrate Assignment table --> Lease Table
	 */

	// Find all Assignment records in the DB
	assignmentScanRes, err := input.dynDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String(input.assignmentTableName),
	})
	if err != nil {
		return fmt.Errorf("failed to make Query API call, %v", err)
	}

	// Unmarshal Assignment records
	assignments := []AccountAssignment{}
	err = dynamodbattribute.UnmarshalListOfMaps(assignmentScanRes.Items, &assignments)
	if err != nil {
		log.Fatalf("failed to unmarshal Assignment result items, %v", err)
	}

	now := time.Now().Unix()
	for _, item := range assignments {
		fmt.Printf("AccountId: %s\n", item.AccountID)

		// Cast the Assignment as a Lease
		lease := Lease{
			AccountID:      item.AccountID,
			PrincipalID:    item.UserID,
			LeaseStatus:    item.AssignmentStatus,
			CreatedOn:      item.CreatedOn,
			LastModifiedOn: now,
		}

		// Marshal the Lease as a DynamoDB Record
		leaseDbItem, err := dynamodbattribute.MarshalMap(lease)
		if err != nil {
			return fmt.Errorf("AccountId: %s error: %v\n", item.AccountID, err)
		}

		// Save the Lease Record
		_, err = input.dynDB.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(input.leaseTableName),
			Item:      leaseDbItem,
		})

		if err != nil {
			return fmt.Errorf("failed to put record: %v", err)
		}
	}

	/**
	 * Migrate Account.Status=Assigned --> Account.Status=Leased
	 */
	// Find all Account records in the DB
	accountScanRes, err := input.dynDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String(input.accountTableName),
	})
	if err != nil {
		return fmt.Errorf("Failed to scan accounts table: %s", err)
	}
	// Filter to accounts with `Status="Assigned"`
	for _, acct := range accountScanRes.Items {
		if *acct["AccountStatus"].S == "Assigned" {
			// Update the AccountStatus to be "Leased"
			_, err = input.dynDB.UpdateItem(&dynamodb.UpdateItemInput{
				TableName: aws.String(input.accountTableName),
				Key: map[string]*dynamodb.AttributeValue{
					"Id": acct["Id"],
				},
				UpdateExpression: aws.String(
					"set AccountStatus=:nextStatus, " +
						"LastModifiedOn=:lastModifiedOn",
				),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":nextStatus": {
						S: aws.String("Leased"),
					},
					":lastModifiedOn": {
						N: aws.String(strconv.FormatInt(now, 10)),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("Failed to update Account status: %s", err)
			}
		}
	}

	return nil
}
