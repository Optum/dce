// This script is intended to find all leased accounts with no active lease

package main

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"os"
)

// main is triggered
func main() {

	// Create DynamoDB Client
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	xlsx, err := excelize.OpenFile("Accounts_NonProd_06012020.xlsx")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read input file
	n := 17
	for i := 1; i < n; i++ {
		accountId, err := xlsx.GetCellValue("Accounts", fmt.Sprintf("A%d", i))
		//fmt.Println(accountId)

		if err != nil {
			fmt.Printf("error reading cell value %d", i)
			continue
		}

		// Check if account exists
		_, err = dynDB.GetItem(
			&dynamodb.GetItemInput{
				// Query in Account Table
				TableName: aws.String("AccountsNonprod"),
				Key: map[string]*dynamodb.AttributeValue{
					"Id": {
						S: aws.String(accountId),
					},
				},
			},
		)

		if err != nil {
			fmt.Printf("No account found %s\n", accountId)
			continue
		}

		leases, err := dynDB.Query(
			&dynamodb.QueryInput{
				TableName:         aws.String("LeasesNonprod"),
				KeyConditions: map[string]*dynamodb.Condition{
					"AccountId": {
						ComparisonOperator: aws.String("EQ"),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								S: aws.String(accountId),
							},
						},
					},
					"LeaseStatus": {
						ComparisonOperator: aws.String("EQ"),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								S: aws.String("Active"),
							},
						},
					},
				},
				ConsistentRead: aws.Bool(true),
			},
		)

		if len(leases.Items) > 1 {
			fmt.Printf("More than one active lease for account %s\n", accountId)
		}

		if (err != nil) || len(leases.Items) == 0 {
			fmt.Printf("No active lease for account %s\n", accountId)
		}

		if len(leases.Items) == 1 {
			fmt.Printf("Active lease for account %s exists\n", accountId)
		}
	}
}
