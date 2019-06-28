// Copyright 2019 Optum, Inc
// db_group_migrate.go
// This script is intended for one time use in the prod deployment of Redbox
// Its sole purpose is to remove the values associated with the deprecated
// "GroupId" field in the dynamo database Account table.
//
// It is intended to be run as a Golang script:
// "go run db_group_migrate.go"
//
// This script requires 3 environment variables to be set for its use:
// "export AWS_CURRENT_REGION=us-east-1"  - The region the database resides in
// "export ACCOUNT_DB=RedboxAccountsProd"  - Name of the Account table
// "export ASSIGNMENT_DB=RedboxAccountAssignmentProd"  - Name of the Assignment table for Accounts

package main

import (
	"strconv"
	"time"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"fmt"
)

type Item struct {
	Id            string
	AccountStatus string
	GroupId       string
}

func main() {
	// Create dynamodb session
	dbSess, err := db.NewFromEnv()
	if err != nil {
		fmt.Print("Failed dbsession create")
	}

	// Build the query input parameters
	params := &dynamodb.ScanInput{
		TableName: aws.String(dbSess.AccountTableName),
	}

	// Make the DynamoDB Query API call
	result, err := dbSess.Client.Scan(params)
	if err != nil {
		fmt.Printf("failed to make Query API call, %v", err)
	}

	items := []Item{}

	// Unmarshal the Items field in the result value to the Item Go type.
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		fmt.Printf("failed to unmarshal Query result items, %v", err)
	}

	// Print out the items returned
	for _, item := range items {
		fmt.Printf("AccountId: %s, AccountStatus: %s\n", item.Id, item.AccountStatus)
		fmt.Printf("\tDeleted Values: %s\n", item.GroupId)
		if len(item.GroupId) != 0 {
			input := &dynamodb.UpdateItemInput{
				TableName: aws.String(dbSess.AccountTableName),
				Key: map[string]*dynamodb.AttributeValue{
					"Id": {
						S: aws.String(item.Id),
					},
				},
				ReturnValues:     aws.String("UPDATED_NEW"),
				UpdateExpression: aws.String("REMOVE GroupId  SET LastModifiedOn = :lastModifiedOn"),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":lastModifiedOn": {
						N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
					},
				},
			}

			_, err := dbSess.Client.UpdateItem(input)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Printf("Deleted %s", item.GroupId)
		}
	}

}
