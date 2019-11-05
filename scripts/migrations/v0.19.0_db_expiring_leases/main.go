package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Optum/dce/pkg/common"
	data "github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	guuid "github.com/google/uuid"
)

// Lease record
type RedboxLease struct {
	AccountID      string `json:"AccountId"`
	PrincipalID    string `json:"PrincipalId"`
	LeaseStatus    string
	CreatedOn      int64
	LastModifiedOn int64
}

// RedboxLeaseMod New Redbox Lease
type RedboxLeaseMod struct {
	AccountID             string `json:"AccountId"`
	PrincipalID           string `json:"PrincipalId"`
	ID                    string `json:"Id"`
	LeaseStatus           string
	ExpiresOn             int64
	CreatedOn             int64
	LastModifiedOn        int64
	LeaseStatusModifiedOn int64
}

type migrationV18Input struct {
	leaseTableName string
	leaseModTime   int64
	dynDB          *dynamodb.DynamoDB
}

// migrationV18 runs main logic
func migrationV18(input *migrationV18Input) (int64, error) {
	// Find all Lease records in the DB
	leaseScanRes, err := input.dynDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String(input.leaseTableName),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to make Query API call, %v", err)
	}

	// Unmarshal Lease records
	leases := []RedboxLease{}
	err = dynamodbattribute.UnmarshalListOfMaps(leaseScanRes.Items, &leases)
	if err != nil {
		log.Fatalf("failed to unmarshal Lease result items, %v", err)
	}

	for _, item := range leases {
		fmt.Printf("AccountId: %s\n", item.AccountID)
		leaseID := guuid.New()
		expiresOn := strconv.Itoa(int(time.Date(2019, time.November, 3, 0, 0, 0, 0, time.Local).Unix()))
		// There are only two statuses for lease now--either Active or Inactive. If
		// it's not Active, any other status is now considered Inactive.
		updatedLeaseStatus := item.LeaseStatus
		if updatedLeaseStatus != string(data.Active) {
			updatedLeaseStatus = string(data.Inactive)
		}
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
				// Set Id to a new unique id
				UpdateExpression: aws.String("set Id=:id, ExpiresOn=:expiresOn, LeaseStatus=:leaseStatus"),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":id": {
						S: aws.String(leaseID.String()),
					},
					":expiresOn": {
						N: aws.String(expiresOn),
					},
					":leaseStatus": {
						S: aws.String(updatedLeaseStatus),
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

	_, err := migrationV18(&migrationV18Input{
		leaseTableName: common.RequireEnv("LEASE_TABLE"),
		leaseModTime:   eseconds,
		dynDB:          dynDB,
	})
	if err != nil {
		log.Fatal(err)
	}
}
