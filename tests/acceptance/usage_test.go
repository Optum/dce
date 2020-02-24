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
	dbSvc.ConsistentRead = true

	// Truncate tables, to make sure we're starting off clean
	truncateUsageTable(t, dbSvc)

	apiURL := tfOut["api_url"].(string)
	// create usage for this lease and account
	expectedUsage := createUsageForInputAmount(t, apiURL, "123456789012", dbSvc, 20.00)

	t.Run("Verify Get Usage By Date Range", func(t *testing.T) {

		// Setup usage dates
		currentTime := time.Now()
		testStartDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -10)

		testutil.Retry(t, 10, 2*time.Second, func(r *testutil.R) {
			// GetUsageByDateRange for testStartDate and 3-days.
			actualUsages, err := dbSvc.GetUsageByDateRange(testStartDate, testStartDate.AddDate(0, 0, 10))
			require.Nil(t, err)

			sort.Slice(expectedUsage, func(i, j int) bool {
				if *expectedUsage[i].StartDate < *expectedUsage[j].StartDate {
					return true
				}
				if *expectedUsage[i].StartDate > *expectedUsage[j].StartDate {
					return false
				}
				return *expectedUsage[i].PrincipalID < *expectedUsage[j].PrincipalID
			})
			sort.Slice(actualUsages, func(i, j int) bool {
				if *actualUsages[i].StartDate < *actualUsages[j].StartDate {
					return true
				}
				if *actualUsages[i].StartDate > *actualUsages[j].StartDate {
					return false
				}
				return *actualUsages[i].PrincipalID < *actualUsages[j].PrincipalID
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
				if *expectedUsage[i].StartDate < *expectedUsage[j].StartDate {
					return true
				}
				if *expectedUsage[i].StartDate > *expectedUsage[j].StartDate {
					return false
				}
				return *expectedUsage[i].PrincipalID < *expectedUsage[j].PrincipalID
			})
			sort.Slice(actualUsage, func(i, j int) bool {
				if *actualUsage[i].StartDate < *actualUsage[j].StartDate {
					return true
				}
				if *actualUsage[i].StartDate > *actualUsage[j].StartDate {
					return false
				}
				return *actualUsage[i].PrincipalID < *actualUsage[j].PrincipalID
			})

			assert.Equal(r, expectedUsage, actualUsage)
		})
	})

	t.Run("GetUsage - When there is a limit filter only", func(t *testing.T) {
		output, err := dbSvc.GetUsage(usage.GetUsageInput{
			Limit: 2,
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(output.Results), "only two usage records should be returned")
	})

	t.Run("GetUsage - When there is a principal ID filter only", func(t *testing.T) {
		output, err := dbSvc.GetUsage(usage.GetUsageInput{
			PrincipalID: "user",
		})
		assert.Nil(t, err)
		assert.Equal(t, len(output.Results), 10, "should only return 10 usage records")
		assert.Equal(t, *output.Results[0].PrincipalID, "user", "should return the usage with the given principal ID")
	})

	t.Run("GetUsage - When there is an account ID filter only", func(t *testing.T) {
		output, err := dbSvc.GetUsage(usage.GetUsageInput{
			AccountID: "123456789012",
		})
		assert.Nil(t, err)
		assert.Equal(t, 10, len(output.Results), "should return the usage with the given account ID")
	})

	t.Run("GetUsage - When there is an start date filter only", func(t *testing.T) {

		currentDate := time.Now()
		testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)

		output, err := dbSvc.GetUsage(usage.GetUsageInput{
			StartDate: testStartDate,
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(output.Results), "should return the usage with the given start date")
	})

	t.Run("GetUsage - When there are limit, start date and principal ID filters", func(t *testing.T) {
		currentDate := time.Now()
		testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
		output, err := getAllUsage(dbSvc, usage.GetUsageInput{
			Limit:       3,
			PrincipalID: "user",
			StartDate:   testStartDate,
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(output), "should only return one usage record")
		assert.Equal(t, "user", *output[0].PrincipalID, "should return the usage with the given principal ID")
	})

	t.Run("GetUsage - When there are no records matching filter", func(t *testing.T) {
		currentDate := time.Now()
		testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
		output, err := getAllUsage(dbSvc, usage.GetUsageInput{
			Limit:       3,
			PrincipalID: "user",
			StartDate:   testStartDate,
			AccountID:   "456",
		})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(output), "should return no usage records")
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

func getAllUsage(dbSvc *usage.DB, input usage.GetUsageInput) ([]*usage.Usage, error) {
	var results []*usage.Usage
	var output usage.GetUsageOutput
	var err error

	for {
		output, err = dbSvc.GetUsage(input)
		if err != nil {
			return nil, err
		}
		results = append(results, output.Results...)
		if len(output.NextKeys) == 0 {
			break
		} else {
			input.StartKeys = output.NextKeys
		}
	}
	return results, nil
}
