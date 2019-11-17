package tests

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"

	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"

	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/tests/acceptance/testutil"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type getItemsRequest struct {
	PrincipalID string
	AccountID   string
}

type getItemsResponseError struct {
	Message string `json:"message"`
}

type getItemsResponseData struct {
	Item string `json:"item"`
}

type getItemsResponseBody struct {
	Result string                 `json:"result"`
	Data   []getItemsResponseData `json:"data"`
	Error  getItemsResponseError  `json:"error"`
}

type getItemsResponseHeaders struct {
	ContentType string `json:"Content-Type"`
}

type getItemsResponse struct {
	StatusCode int                     `json:"statusCode"`
	Headers    getItemsResponseHeaders `json:"headers"`
	Body       getItemsResponseBody    `json:"body"`
}

func TestUpdateLeaseStatusLambda(t *testing.T) {

	// Load Terraform outputs
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)
	apiURL := tfOut["api_url"].(string)

	// Configure the DB service
	// Configure the S3 Service
	awsSession, err := session.NewSession(
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)))
	require.Nil(t, err)
	require.Nil(t, err)
	dbSvc := db.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["accounts_table_name"].(string),
		tfOut["leases_table_name"].(string),
		7,
	)

	// Configure the usage service
	usageSvc := usage.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["usage_table_name"].(string),
	)

	// Create Lambda service client
	lambdaClient := lambda.New(awsSession)

	// Make sure the DB is clean
	truncateDBTables(t, dbSvc)
	truncateUsageTable(t, usageSvc)

	// Cleanup the DB when we're done
	defer truncateDBTables(t, dbSvc)
	defer truncateUsageTable(t, usageSvc)

	// Create an adminRole for the account
	adminRoleRes := createAdminRole(t, awsSession)
	accountID := adminRoleRes.accountID
	adminRoleArn := adminRoleRes.adminRoleArn
	//adminRoleArn := "arn:aws:iam::391501768339:role/AWS_391501768339_Admins"
	principalID := "user"

	// Add the current account to the account pool
	apiRequest(t, &apiRequestInput{
		method: "POST",
		url:    apiURL + "/accounts",
		json: createAccountRequest{
			ID:           accountID,
			AdminRoleArn: adminRoleArn,
		},
		maxAttempts: 15,
		f: func(r *testutil.R, apiResp *apiResponse) {
			assert.Equal(r, 201, apiResp.StatusCode)
		},
	})

	// Update Account status to ready
	_,err = dbSvc.TransitionAccountStatus(
		accountID,
		db.NotReady, db.Ready,
	)
	require.Nil(t, err)

	// Create a lease for above account
	apiRequest(t, &apiRequestInput{
		method: "POST",
		url:    apiURL + "/leases",
		json: inputLeaseRequest{
			PrincipalID: principalID,
			BudgetAmount:20.00,

		},
		maxAttempts: 15,
		f: func(r *testutil.R, apiResp *apiResponse) {
			assert.Equal(r, 201, apiResp.StatusCode)
		},
	})

	// create usage
	createTestUsage(t, apiURL, accountID, usageSvc)

	// Test update_lease_status lambda
	t.Run("update_lease_status", func(t *testing.T) {
		t.Run("Should run update_lease_status lambda successfully", func(t *testing.T) {

			// Get the 10 most recent items
			request := getItemsRequest{principalID, accountID}

			payload, err := json.Marshal(request)
			require.Nil(t, err)

			result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
			require.Nil(t, err)

			var resp getItemsResponse

			err = json.Unmarshal(result.Payload, &resp)
			require.Nil(t, err)

			fmt.Println(resp)

		})
	})

}

func createTestUsage(t *testing.T, apiURL string, accountID string, usageSvc usage.Service) {
	// Create usage
	// Setup usage dates
	const ttl int = 3
	currentDate := time.Now()
	testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
	testEndDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 59, time.UTC)

	// Create mock usage
	var expectedUsages []*usage.Usage

	usageStartDate := testStartDate
	usageEndDate := testEndDate
	startDate := testStartDate
	endDate := testEndDate

	timeToLive := startDate.AddDate(0, 0, ttl)

	var testPrincipalID = "user"
	var testAccountID = accountID

	for i := 1; i <= 5; i++ {

		input := usage.Usage{
			PrincipalID:  testPrincipalID,
			AccountID:    testAccountID,
			StartDate:    startDate.Unix(),
			EndDate:      endDate.Unix(),
			CostAmount:   20.00,
			CostCurrency: "USD",
			TimeToLive:   timeToLive.Unix(),
		}
		err := usageSvc.PutUsage(input)
		require.Nil(t, err)

		expectedUsages = append(expectedUsages, &input)

		usageEndDate = endDate
		startDate = startDate.AddDate(0, 0, -1)
		endDate = endDate.AddDate(0, 0, -1)
	}

	queryString := fmt.Sprintf("/usage?startDate=%d&endDate=%d", usageStartDate.Unix(), usageEndDate.Unix())

	testutil.Retry(t, 10, 10*time.Millisecond, func(r *testutil.R) {

		resp := apiRequest(t, &apiRequestInput{
			method: "GET",
			url:    apiURL + queryString,
			json:   nil,
		})

		// Verify response code
		assert.Equal(r, http.StatusOK, resp.StatusCode)

		// Parse response json
		data := parseResponseArrayJSON(t, resp)

		//Verify response json
		if len(data) > 0 && data[0] != nil {
			usageJSON := data[0]
			assert.Equal(r, "TestUser1", usageJSON["principalId"].(string))
			assert.Equal(r, "TestAcct1", usageJSON["accountId"].(string))
			assert.Equal(r, 100.00, usageJSON["costAmount"].(float64))
		}
	})
}
