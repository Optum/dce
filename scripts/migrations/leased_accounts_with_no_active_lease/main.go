// This script is intended to find all leased accounts with no active lease

package main

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/lease"
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

	xlsx, err := excelize.OpenFile("LeasedAccounts05132020.xlsx")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read input file
	n := 100
	for i := 1; i < n; i++ {
		accountId := xlsx.GetCellValue("Accounts", fmt.Sprintf("A%d", i))
		//fmt.Println(accountId)

		// Check if account exists
		_, err := input.dynDB.GetItem(
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

		if err != nil {
			fmt.Printf("No account found %s", accountId)
			continue
		}

		var i = data.Lease{
			Limit:          25,
			ConsistentRead: true,
			TableName:      "Leases",
			DynamoDB:       dynDB,
		}

		var query = lease.Lease {
			Status: lease.StatusActive.StatusPtr(),
			AccountID: aws.String(accountId),
		}
		leases, err := i.List(&query)

		fmt.Println(err)

		if err != nil {
			continue
		}

		if len(*leases) > 1 {
			fmt.Printf("More than one active lease for account %s", accountId)
		}

		if len(*leases) == 0 {
			fmt.Printf("No active lease for account %s", accountId)
		}

		if len(*leases) == 1 {
			fmt.Printf("Active lease for account %s exists", accountId)
		}
	}
}
