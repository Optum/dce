package tests

import (
	"sort"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/tests/acceptance/testutil"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
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
		tfOut["usage_table_name"].(string),
		"StartDate",
		"PrincipalId",
	)

	// For testing purposes support consistent reads
	dbSvc.ConsistendRead = true

	// ttl is set to 3-days
	const ttl int = 3

	// Truncate tables, to make sure we're starting off clean
	truncateUsageTable(t, dbSvc)

	apiURL := tfOut["api_url"].(string)
	// create usage for this lease and account
	expectedUsage := createUsageForInputAmount(t, apiURL, "123", dbSvc, 20.00)

	t.Run("Verify Get Usage By Date Range", func(t *testing.T) {

		// Setup usage dates
		currentTime := time.Now()
		testStartDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -10)

		testutil.Retry(t, 10, 2*time.Second, func(r *testutil.R) {
			// GetUsageByDateRange for testStartDate and 3-days.
			actualUsages, err := dbSvc.GetUsageByDateRange(testStartDate, testStartDate.AddDate(0, 0, 10))
			require.Nil(t, err)

			sort.Slice(expectedUsage, func(i, j int) bool {
				if expectedUsage[i].StartDate < expectedUsage[j].StartDate {
					return true
				}
				if expectedUsage[i].StartDate > expectedUsage[j].StartDate {
					return false
				}
				return expectedUsage[i].PrincipalID < expectedUsage[j].PrincipalID
			})
			sort.Slice(actualUsages, func(i, j int) bool {
				if actualUsages[i].StartDate < actualUsages[j].StartDate {
					return true
				}
				if actualUsages[i].StartDate > actualUsages[j].StartDate {
					return false
				}
				return actualUsages[i].PrincipalID < actualUsages[j].PrincipalID
			})

			assert.Equal(r, expectedUsage, actualUsages)
		})
	})

	t.Run("Verify Get Usage By PrincipalId", func(t *testing.T) {

		// Setup usage dates
		currentTime := time.Now()
		testStartDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -9)

		testutil.Retry(t, 10, 2*time.Second, func(r *testutil.R) {

			actualUsage, err := dbSvc.GetUsageByPrincipal(testStartDate, "user")
			require.Nil(t, err)

			sort.Slice(expectedUsage, func(i, j int) bool {
				if expectedUsage[i].StartDate < expectedUsage[j].StartDate {
					return true
				}
				if expectedUsage[i].StartDate > expectedUsage[j].StartDate {
					return false
				}
				return expectedUsage[i].PrincipalID < expectedUsage[j].PrincipalID
			})
			sort.Slice(actualUsage, func(i, j int) bool {
				if actualUsage[i].StartDate < actualUsage[j].StartDate {
					return true
				}
				if actualUsage[i].StartDate > actualUsage[j].StartDate {
					return false
				}
				return actualUsage[i].PrincipalID < actualUsage[j].PrincipalID
			})

			assert.Equal(r, expectedUsage, actualUsage)
		})
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
			TableName:      aws.String(dbSvc.UsageTableName),
			ConsistentRead: aws.Bool(true),
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
					"StartDate":   item["StartDate"],
					"PrincipalId": item["PrincipalId"],
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
