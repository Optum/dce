// dbAddLeaseModOn.go
// This script is intended for one time use in the prod deployment of Dce
// Its sole purpose is to add LeaseStatusModifiedOn field, with current epoch seconds value,
// to the Prod Leases table for AWS_Dce
//
// It is intended to be run as a Golang script:
// "go run dbAddLeaseModOn.go"
//
// This script requires 2 environment variables to be set for its use:
// "export AWS_CURRENT_REGION=us-east-1"  - The region the database resides in
// "export LEASE_DB=DceLeasetProd"  - Name of the Lease table for Accounts. This is the new table to which items from SOURCE_DB will be added to

package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Optum/Dce/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// DceLease record
type DceLease struct {
	AccountID      string `json:"AccountId"`
	PrincipalID    string `json:"PrincipalId"`
	LeaseStatus    string
	CreatedOn      int64
	LastModifiedOn int64
}

type DceLeaseMod struct {
	AccountID             string `json:"AccountId"`
	PrincipalID           string `json:"PrincipalId"`
	LeaseStatus           string
	CreatedOn             int64
	LastModifiedOn        int64
	LeaseStatusModifiedOn int64
}

type migrationV11Input struct {
	leaseTableName string
	leaseModTime   int64
	dynDB          *dynamodb.DynamoDB
}

// migrationV11 runs main logic
func migrationV11(input *migrationV11Input) (int64, error) {
	// Find all Lease records in the DB
	leaseScanRes, err := input.dynDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String(input.leaseTableName),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to make Query API call, %v", err)
	}

	// Unmarshal Lease records
	eseconds := strconv.FormatInt(time.Now().Unix(), 10)
	leases := []DceLease{}
	err = dynamodbattribute.UnmarshalListOfMaps(leaseScanRes.Items, &leases)
	if err != nil {
		log.Fatalf("failed to unmarshal Lease result items, %v", err)
	}

	for _, item := range leases {
		fmt.Printf("AccountId: %s\n", item.AccountID)

		result, err := input.dynDB.UpdateItem(
			&dynamodb.UpdateItemInput{
				// Query in Lease Table
				TableName: aws.String(input.leaseTableName),
				// Find Lease for the requested accountId
				Key: map[string]*dynamodb.AttributeValue{
					"AccountId": {
						S: aws.String(item.AccountID),
					},
					"PrincipalId": {
						S: aws.String(item.PrincipalID),
					},
				},
				// Set Status="Active"
				UpdateExpression: aws.String("set LeaseStatusModifiedOn=:leaseStatusModifiedOn, LastModifiedOn=:lastModifiedOn"),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":leaseStatusModifiedOn": {
						N: aws.String(eseconds),
					},
					":lastModifiedOn": {
						N: aws.String(eseconds),
					},
				},
				// Return the updated record
				ReturnValues: aws.String("ALL_NEW"),
			},
		)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == "ConditionalCheckFailedException" {
					return 0, fmt.Errorf(
						"unable to update lease mod date for %v/%v",
						item.AccountID,
						item.PrincipalID,
					)
				}
			}
			return 0, err
		}
		fmt.Println(result)
	}
	return input.leaseModTime, nil
}

// func main() {
// 	fmt.Println(time.Now().Unix())

// main is triggered
func main() {

	// Create DynamoDB Client
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	// Set a time we can compare to
	eseconds := time.Now().Unix()

	_, err := migrationV11(&migrationV11Input{
		leaseTableName: common.RequireEnv("LEASE_TABLE"),
		leaseModTime:   eseconds,
		dynDB:          dynDB,
	})
	if err != nil {
		log.Fatal(err)
	}
}
