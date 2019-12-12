package tests

import (
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	"encoding/json"
	"fmt"

	"github.com/stretchr/testify/assert"

	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/tests/acceptance/testutil"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

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
		"StartDate",
		"PrincipalId",
	)

	// Create Lambda service client
	lambdaClient := lambda.New(awsSession)

	// Create an adminRole for the account
	var adminRoleName = "dce-api-test-admin-role-updateleasestatus-" + fmt.Sprintf("%v", time.Now().Unix())
	adminRoleRes := createAdminRole(t, awsSession, adminRoleName)
	defer deleteAdminRole()

	t.Run("Not exceeded lease budget result in Active lease with reason Active.", func(t *testing.T) {

		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		truncateUsageTable(t, usageSvc)

		// Cleanup the DB after test is done
		defer truncateDBTables(t, dbSvc)
		defer truncateUsageTable(t, usageSvc)

		accountID := adminRoleRes.accountID
		adminRoleArn := adminRoleRes.adminRoleArn
		principalID := "user"

		// Add current account to the account pool
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
		_, err = dbSvc.TransitionAccountStatus(
			accountID,
			db.NotReady, db.Ready,
		)
		require.Nil(t, err)

		// Create a lease for above created account
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/leases",
			json: inputLeaseRequest{
				PrincipalID:  principalID,
				BudgetAmount: 200.00,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// Invoke update_lease_status lambda
		request := db.Lease{
			AccountID:             accountID,
			PrincipalID:           principalID,
			LeaseStatus:           db.LeaseStatus("Active"),
			LeaseStatusModifiedOn: time.Now().AddDate(0, 0, -1).Unix(),
			ExpiresOn:             time.Now().AddDate(0, 0, 3).Unix(),
			BudgetAmount:          200.00,
		}
		payload, err := json.Marshal(request)
		require.Nil(t, err)

		result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
		require.Nil(t, err)

		var resp getItemsResponse
		err = json.Unmarshal(result.Payload, &resp)
		require.Nil(t, err)

		// Check lease status is active
		lease, err := dbSvc.GetLease(accountID, "user")
		require.Nil(t, err)
		require.Equal(t, db.LeaseStatus("Active"), lease.LeaseStatus)
		require.Equal(t, db.LeaseStatusReason("Active"), lease.LeaseStatusReason)
	})

	t.Run("Expired lease result in Inactive lease with reason Expired.", func(t *testing.T) {

		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		truncateUsageTable(t, usageSvc)

		// Cleanup the DB after test is done
		defer truncateDBTables(t, dbSvc)
		defer truncateUsageTable(t, usageSvc)

		accountID := adminRoleRes.accountID
		adminRoleArn := adminRoleRes.adminRoleArn
		principalID := "user"

		// Add current account to the account pool
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
		_, err = dbSvc.TransitionAccountStatus(
			accountID,
			db.NotReady, db.Ready,
		)
		require.Nil(t, err)

		// Create a lease for above created account
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/leases",
			json: inputLeaseRequest{
				PrincipalID:  principalID,
				BudgetAmount: 200.00,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// Invoke update_lease_status lambda
		request := db.Lease{
			AccountID:             accountID,
			PrincipalID:           principalID,
			LeaseStatus:           db.LeaseStatus("Active"),
			LeaseStatusModifiedOn: time.Now().AddDate(0, 0, -1).Unix(),
			ExpiresOn:             time.Now().AddDate(0, 0, -1).Unix(),
			BudgetAmount:          200.00,
		}
		payload, err := json.Marshal(request)
		require.Nil(t, err)

		result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
		require.Nil(t, err)

		var resp getItemsResponse
		err = json.Unmarshal(result.Payload, &resp)
		require.Nil(t, err)

		// Check lease status is active
		lease, err := dbSvc.GetLease(accountID, "user")
		require.Nil(t, err)
		require.Equal(t, db.LeaseStatus("Inactive"), lease.LeaseStatus)
		require.Equal(t, db.LeaseStatusReason("Expired"), lease.LeaseStatusReason)
	})

	t.Run("Exceeded lease budget result in Inactive lease with reason OverBudget.", func(t *testing.T) {

		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		truncateUsageTable(t, usageSvc)

		// Cleanup the DB when test execution is done
		defer truncateDBTables(t, dbSvc)
		defer truncateUsageTable(t, usageSvc)

		accountID := adminRoleRes.accountID
		adminRoleArn := adminRoleRes.adminRoleArn
		principalID := "user"

		// Add current account to the account pool
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
		_, err = dbSvc.TransitionAccountStatus(
			accountID,
			db.NotReady, db.Ready,
		)
		require.Nil(t, err)

		// Create a lease for above created account
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/leases",
			json: inputLeaseRequest{
				PrincipalID:  principalID,
				BudgetAmount: 199.00,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// create usage for this lease and account
		createUsageForInputAmount(t, apiURL, accountID, usageSvc, 200.00)

		// Invoke update_lease_status lambda
		request := db.Lease{
			AccountID:             accountID,
			PrincipalID:           principalID,
			LeaseStatus:           db.LeaseStatus("Active"),
			LeaseStatusModifiedOn: time.Now().AddDate(0, 0, -1).Unix(),
			ExpiresOn:             time.Now().AddDate(0, 0, 3).Unix(),
			BudgetAmount:          199.00,
		}
		payload, err := json.Marshal(request)
		require.Nil(t, err)

		result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
		require.Nil(t, err)

		var resp getItemsResponse
		err = json.Unmarshal(result.Payload, &resp)
		require.Nil(t, err)

		// Check lease status is inactive due to lease over budget
		lease, err := dbSvc.GetLease(accountID, "user")
		require.Nil(t, err)
		require.Equal(t, db.LeaseStatus("Inactive"), lease.LeaseStatus)
		require.Equal(t, db.LeaseStatusReason("OverBudget"), lease.LeaseStatusReason)
	})

	t.Run("Exceeded principal budget result in Inactive lease with reason OverPrincipalBudget.", func(t *testing.T) {

		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		truncateUsageTable(t, usageSvc)

		// Cleanup the DB when test execution is done
		defer truncateDBTables(t, dbSvc)
		defer truncateUsageTable(t, usageSvc)

		accountID := adminRoleRes.accountID
		adminRoleArn := adminRoleRes.adminRoleArn
		principalID := "user"

		// Add current account to the account pool
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
		_, err = dbSvc.TransitionAccountStatus(
			accountID,
			db.NotReady, db.Ready,
		)
		require.Nil(t, err)

		// Create a lease for above created account
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/leases",
			json: inputLeaseRequest{
				PrincipalID:  principalID,
				BudgetAmount: 300.00,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// create usage for this lease and account
		createUsageForInputAmount(t, apiURL, accountID, usageSvc, 2000.00)

		// Invoke update_lease_status lambda
		request := db.Lease{
			AccountID:             accountID,
			PrincipalID:           principalID,
			LeaseStatus:           db.LeaseStatus("Active"),
			LeaseStatusModifiedOn: time.Now().Unix(),
			ExpiresOn:             time.Now().AddDate(0, 0, 3).Unix(),
			BudgetAmount:          300.00,
		}
		payload, err := json.Marshal(request)
		require.Nil(t, err)

		result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
		require.Nil(t, err)

		var resp getItemsResponse
		err = json.Unmarshal(result.Payload, &resp)
		require.Nil(t, err)

		// Check lease status is inactive due to lease over budget
		lease, err := dbSvc.GetLease(accountID, "user")
		require.Nil(t, err)
		require.Equal(t, db.LeaseStatus("Inactive"), lease.LeaseStatus)
		require.Equal(t, db.LeaseStatusReason("OverPrincipalBudget"), lease.LeaseStatusReason)
	})

	t.Run("Exceeded both lease and principal budget result in Inactive lease with reason OverBudget.", func(t *testing.T) {

		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		truncateUsageTable(t, usageSvc)

		// Cleanup the DB when test execution is done
		defer truncateDBTables(t, dbSvc)
		defer truncateUsageTable(t, usageSvc)

		accountID := adminRoleRes.accountID
		adminRoleArn := adminRoleRes.adminRoleArn
		principalID := "user"

		// Add current account to the account pool
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
		_, err = dbSvc.TransitionAccountStatus(
			accountID,
			db.NotReady, db.Ready,
		)
		require.Nil(t, err)

		// Create a lease for above created account
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/leases",
			json: inputLeaseRequest{
				PrincipalID:  principalID,
				BudgetAmount: 300.00,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// create usage for this lease and account
		createUsageForInputAmount(t, apiURL, accountID, usageSvc, 2000.00)

		// Invoke update_lease_status lambda
		request := db.Lease{
			AccountID:             accountID,
			PrincipalID:           principalID,
			LeaseStatus:           db.LeaseStatus("Active"),
			LeaseStatusModifiedOn: time.Now().AddDate(0, 0, -3).Unix(),
			ExpiresOn:             time.Now().AddDate(0, 0, 3).Unix(),
			BudgetAmount:          300.00,
		}
		payload, err := json.Marshal(request)
		require.Nil(t, err)

		result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
		require.Nil(t, err)

		var resp getItemsResponse
		err = json.Unmarshal(result.Payload, &resp)
		require.Nil(t, err)

		// Check lease status is inactive due to lease over budget
		lease, err := dbSvc.GetLease(accountID, "user")
		require.Nil(t, err)
		require.Equal(t, db.LeaseStatus("Inactive"), lease.LeaseStatus)
		require.Equal(t, db.LeaseStatusReason("OverBudget"), lease.LeaseStatusReason)
	})

}

func createUsageForInputAmount(t *testing.T, apiURL string, accountID string, usageSvc usage.Service, costAmount float64) []*usage.Usage {
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

	for i := 1; i <= 10; i++ {

		input := usage.Usage{
			PrincipalID:  testPrincipalID,
			AccountID:    testAccountID,
			StartDate:    startDate.Unix(),
			EndDate:      endDate.Unix(),
			CostAmount:   costAmount,
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
		log.Printf("%+v\n", resp)
		data := parseResponseArrayJSON(t, resp)

		//Verify response json
		if len(data) > 0 && data[0] != nil {
			usageJSON := data[0]
			assert.Equal(r, "TestUser1", usageJSON["principalId"].(string))
			assert.Equal(r, "TestAcct1", usageJSON["accountId"].(string))
			assert.Equal(r, 1000.00, usageJSON["costAmount"].(float64))
		}
	})

	return expectedUsages
}
