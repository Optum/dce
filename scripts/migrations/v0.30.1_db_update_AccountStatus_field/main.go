// This script is intended for one time use to update account status for selected accounts to DeleteReady

package main

import (
	"fmt"
	"os"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type UpdateStatusInput struct {
	accountId        string
	accountTableName string
	dynDB            *dynamodb.DynamoDB
}

// updateStatus runs main logic
func updateStatus(input *UpdateStatusInput) error {

	// Check if account exists
	res, err := input.dynDB.GetItem(
		&dynamodb.GetItemInput{
			// Query in Account Table
			TableName: aws.String(input.accountTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(input.accountId),
				},
			},
		},
	)

	if err != nil || len(res.Item) == 0 {
		return fmt.Errorf("get failed for account %q", input.accountId)
	}

	// Update Account record
	_, err = input.dynDB.UpdateItem(
		&dynamodb.UpdateItemInput{
			// Query in Account Table
			TableName: aws.String(input.accountTableName),
			// Find Account for the requested accountId
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(input.accountId),
				},
			},
			// Set Status="DeleteReady"
			UpdateExpression: aws.String("set AccountStatus=:accountStatus"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":accountStatus": {
					S: aws.String("DeleteReady"),
				},
			},
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Account updated %q", input.accountId)

	return nil
}

// main is triggered
func main() {

	// Create DynamoDB Client
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	xlsx, err := excelize.OpenFile("accountstodelete.xlsx")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read input file
	n := 368
	for i := 1; i < n; i++ {
		accountId := xlsx.GetCellValue("To Decom", fmt.Sprintf("D%d", i))
		//fmt.Println(accountId)

		err = updateStatus(&UpdateStatusInput{
			accountId:        accountId,
			accountTableName: "Accounts",
			dynDB:            dynDB,
		})
		fmt.Println(err)
	}
}
