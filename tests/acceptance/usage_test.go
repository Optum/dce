package tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Optum/Redbox/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

func TestUsageDb(t *testing.T) {
	// Load Terraform outputs
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	// Configure the Usage DB service
	awsSession, err := session.NewSession()
	require.Nil(t, err)
	dbSvc := usage.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["usage_cache_table_name"].(string),
	)

	// Truncate tables, to make sure we're starting off clean
	//truncateUsageTable(t, dbSvc)

	t.Run("PutUsage / GetUsageByDaterange", func(t *testing.T) {

		// Cleanup table on completion
		//defer truncateUsageTable(t, dbSvc)

		// Create mock usages
		expectedUsages := []*usage.Usage{}
		for a := 1; a <= 10; a++ {

			startDate := 1564790400
			endDate := 1564876799
			ttl := startDate + (86400 * 3)

			var testPrinciplaID []string
			var testAccountID []string

			testPrinciplaID = append(testPrinciplaID, "Test")
			testPrinciplaID = append(testPrinciplaID, strconv.Itoa(a))

			testAccountID = append(testAccountID, "Acct")
			testAccountID = append(testAccountID, strconv.Itoa(a))

			for i := 1; i <= 30; i++ {

				input := usage.Usage{
					PrincipalID:  strings.Join(testPrinciplaID, ""),
					AccountID:    strings.Join(testAccountID, ""),
					StartDate:    startDate,
					EndDate:      endDate,
					CostAmount:   23.00,
					CostCurrency: "USD",
					TimeToExist:  ttl,
				}
				err = dbSvc.PutUsage(input)
				require.Nil(t, err)
				if startDate >= 1564790400 || startDate <= 1564876800 {
					expectedUsages = append(expectedUsages, &input)
				}

				startDate = startDate + 86400
				endDate = endDate + 86400
			}
		}

		// GetUsageByDaterange for startDate 1564790400 and 2-days.
		startDate := 1564790400
		days := 2
		actualUsages, err := dbSvc.GetUsageByDaterange(startDate, days)
		require.Nil(t, err)
		require.Equal(t, expectedUsages, actualUsages)
	})

}

// Remove all records from the Usage table
func truncateUsageTable(t *testing.T, dbSvc *usage.DB) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Usage table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName: aws.String(dbSvc.UsageTableName),
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
					"StartDate": item["StartDate"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.Client.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dbSvc.UsageTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
}
