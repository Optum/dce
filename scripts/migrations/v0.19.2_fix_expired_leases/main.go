/*
Migration for v0.19.2

v0.19.0 included a bug in the `update_lease_status` lambda
which set active leases to `LeaseStatus=Inactive, LeaseStatusReason=Expired`
even if the `ExpiresOn` property was in the future.

This migration identifies the wrongly expired leases, and sets them back to
`LeaseStatus=Active` again.
*/
package main

import (
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"os"
)

func main() {
	// Create DynamoDB Client
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	// Find all Inactive item records
	// which were wrongfully expired
	leaseTableName := os.Getenv("LEASE_TABLE_NAME")
	leaseRes, err := dynDB.Scan(&dynamodb.ScanInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {S: aws.String("Inactive")},
			":reason": {S: aws.String("Expired")},
			// Hardcoded ExpiresOn from v0.19.0 migration
			":expires": {N: aws.String("1572739200")},
		},
		FilterExpression: aws.String(
			"LeaseStatus=:status AND " +
				"LeaseStatusReason=:reason AND " +
				"ExpiresOn=:expires",
		),
		TableName: aws.String(leaseTableName),
	})
	if err != nil {
		log.Fatalf("Failed to scan: %v", err)
	}

	// Unmarshal Lease records
	leases := []db.RedboxLease{}
	err = dynamodbattribute.UnmarshalListOfMaps(leaseRes.Items, &leases)
	if err != nil {
		log.Fatalf("failed to unmarshal Lease result items, %v", err)
	}

	for _, item := range leases {
		// Mark the record as Active again
		_, err = dynDB.UpdateItem(&dynamodb.UpdateItemInput{
			TableName: aws.String(leaseTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: aws.String(item.AccountID),
				},
				"PrincipalId": {
					S: aws.String(item.PrincipalID),
				},
			},
			UpdateExpression: aws.String(
				"set LeaseStatus=:status, LeaseStatusReason=:reason",
			),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":status": {S: aws.String("Active")},
				":reason": {S: aws.String("Active")},
			},
		})
		if err != nil {
			log.Printf("Failed to update lease for %s/%s: %v", item.PrincipalID, item.AccountID, err)
			continue
		}
		log.Printf("Set lease to Active for %s/%s", item.PrincipalID, item.AccountID)
	}
}
