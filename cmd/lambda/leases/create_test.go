package main

import (
	"context"
	"encoding/json"
	"errors"
	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"

	accountmocks "github.com/Optum/dce/pkg/account/accountiface/mocks"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	leasemocks "github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWhenCreateSuccess(t *testing.T) {
	tests := []struct {
		// Test case name
		name string

		// Mock API user
		user *api.User

		// JSON body to send to API controller
		requestBody map[string]interface{}

		// Lease we expect to be created in DB
		expectedLeaseToCreate *lease.Lease

		// Expected HTTP Response code from API controller
		expectedResponseCode int

		// Expected HTTP Response JSON body
		expectedResponseBody map[string]interface{}

		// List of ready account to return from DB
		mockReadyAccounts *account.Accounts

		// Lease to return, when checking for existing lease
		// for this principal/account
		getExistingLeases *lease.Leases

		// Errors to return from DB operations
		getExistingLeasesErr error
		listAccountError     error
		retUpdateErr         error
		retCreateErr         error
	}{
		{
			name: "Should create a new lease",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User1",
				"budgetAmount": 200.00,
			},
			mockReadyAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			expectedLeaseToCreate: &lease.Lease{
				PrincipalID:  ptrString("User1"),
				BudgetAmount: ptrFloat64(200),
				AccountID:    ptrString("1234567890"),
			},
			expectedResponseCode: http.StatusCreated,
			expectedResponseBody: map[string]interface{}{
				"id":                "mock-lease-id",
				"principalId":       "User1",
				"budgetAmount":      200.00,
				"accountId":         "1234567890",
				"leaseStatus":       "Active",
				"leaseStatusReason": "Active",
			},
		},
		{
			name: "Should reactivate an existing inactive lease, " +
				"for the sam principal ID / account ID",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User1",
				"budgetAmount": 200.00,
			},
			mockReadyAccounts: &account.Accounts{
				{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			// Return an existing inactive lease for this same
			// principalID / accountID combo
			getExistingLeases: &lease.Leases{
				{
					AccountID:      ptrString("1234567890"),
					PrincipalID:    ptrString("User1"),
					Status:         lease.StatusInactive.StatusPtr(),
					CreatedOn:      ptrInt64(100),
					LastModifiedOn: ptrInt64(200),
				},
			},
			expectedLeaseToCreate: &lease.Lease{
				PrincipalID:  ptrString("User1"),
				BudgetAmount: ptrFloat64(200),
				AccountID:    ptrString("1234567890"),
				// Should pass timestamps of existing lease
				// to data svc.
				// This is to fignal that we are updating (not creating) a lease record
				CreatedOn:      ptrInt64(100),
				LastModifiedOn: ptrInt64(200),
			},
			expectedResponseCode: http.StatusCreated,
			expectedResponseBody: map[string]interface{}{
				"id":                "mock-lease-id",
				"principalId":       "User1",
				"budgetAmount":      200.00,
				"accountId":         "1234567890",
				"leaseStatus":       "Active",
				"leaseStatusReason": "Active",
				"createdOn":         100.00,
				"lastModifiedOn":    200.00,
			},
		},
		{
			name: "Should return a client error if principalId is missing",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			// Send request, missing required "principalId" field
			requestBody: map[string]interface{}{
				"budgetAmount": 200.00,
			},
			// Should return a 400 error
			expectedResponseCode: http.StatusBadRequest,
			expectedResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "invalid request parameters: missing principalId",
					"code":    "ClientError",
				},
			},
		},
		{
			name: "Should return a client error for budget amount as string",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			requestBody: map[string]interface{}{
				"budgetAmount": "as much as you'd like",
			},
			// Should return a 400 error
			expectedResponseCode: http.StatusBadRequest,
			expectedResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "invalid request parameters",
					"code":    "ClientError",
				},
			},
		},
		{
			name: "Should return auth error if non-admin user creates a lease for another user",
			user: &api.User{
				Username: "User1",
				Role:     api.UserGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User2",
				"budgetAmount": 200.00,
			},
			// Should return a 400 error
			expectedResponseCode: http.StatusUnauthorized,
			expectedResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "User [User1] with role: [User] attempted to act on a lease for [User2], but was not authorized",
					"code":    "UnauthorizedError",
				},
			},
		},
		{
			name: "Should allow non-admin user to create a lease for themselves",
			user: &api.User{
				Username: "User1",
				Role:     api.UserGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User1",
				"budgetAmount": 200.00,
			},
			mockReadyAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			expectedLeaseToCreate: &lease.Lease{
				PrincipalID:  ptrString("User1"),
				BudgetAmount: ptrFloat64(200),
				AccountID:    ptrString("1234567890"),
			},
			expectedResponseCode: http.StatusCreated,
			expectedResponseBody: map[string]interface{}{
				"id":                "mock-lease-id",
				"principalId":       "User1",
				"budgetAmount":      200.00,
				"accountId":         "1234567890",
				"leaseStatus":       "Active",
				"leaseStatusReason": "Active",
			},
		},
		{
			name: "Should return ServerError if accounts DB query fails",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User1",
				"budgetAmount": 200.00,
			},
			listAccountError:     errors.New("DB request failed"),
			expectedResponseCode: http.StatusInternalServerError,
			expectedResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "ServerError",
					"message": "unknown error",
				},
			},
		},
		{
			name: "Should return ServerError if lease DB update fails",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User1",
				"budgetAmount": 200.00,
			},
			mockReadyAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			expectedLeaseToCreate: &lease.Lease{
				PrincipalID:  ptrString("User1"),
				BudgetAmount: ptrFloat64(200),
				AccountID:    ptrString("1234567890"),
			},
			retUpdateErr:         errors.New("DB request failed"),
			expectedResponseCode: http.StatusInternalServerError,
			expectedResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "ServerError",
					"message": "unknown error",
				},
			},
		},
		{
			name: "Should return a ServerError if no accounts are available",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			requestBody: map[string]interface{}{
				"principalId":  "User1",
				"budgetAmount": 200.00,
			},
			// no accounts ready and available
			mockReadyAccounts:    &account.Accounts{},
			expectedResponseCode: http.StatusInternalServerError,
			expectedResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "ServerError",
					"message": "No Available accounts at this moment",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare Mocks
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := leasemocks.Servicer{}
			accountSvc := accountmocks.Servicer{}
			userDetailSvc := apiMocks.UserDetailer{}

			// Return a mock API user
			userDetailSvc.On("GetUser", mock.Anything).Return(tt.user)

			// Mock a list of ready accounts from the DB
			accountSvc.
				On("List", &account.Account{
					Status: account.StatusReady.StatusPtr(),
				}).
				Return(tt.mockReadyAccounts, tt.listAccountError)

			if tt.mockReadyAccounts != nil && len(*tt.mockReadyAccounts) > 0 {
				// Should mark the account as leased
				firstReadyAccount := (*tt.mockReadyAccounts)[0]
				accountSvc.
					On("Update", *firstReadyAccount.ID, mock.MatchedBy(func(acct *account.Account) bool {
						assert.Equal(t, account.StatusLeased.StatusPtr(), acct.Status)
						return true
					})).
					Return(func(id string, acct *account.Account) *account.Account {
						// Return the updated account
						return acct
					}, tt.retUpdateErr)

				// Should query for existing inactive leases for the same account / principal
				leaseSvc.
					On("List", &lease.Lease{
						AccountID:   firstReadyAccount.ID,
						PrincipalID: ptrString(tt.requestBody["principalId"].(string)),
						Status:      lease.StatusInactive.StatusPtr(),
					}).
					Return(tt.getExistingLeases, tt.getExistingLeasesErr)
			}

			// Should create a lease
			var leaseFromDB *lease.Lease
			leaseSvc.
				On("Create", tt.expectedLeaseToCreate).
				Return(func(leaseInput *lease.Lease) *lease.Lease {
					// Create a copy of the lease,
					// with Status=Active
					leaseFromDB = leaseInput
					leaseFromDB.ID = ptrString("mock-lease-id")
					leaseFromDB.Status = lease.StatusActive.StatusPtr()
					leaseFromDB.StatusReason = lease.StatusReasonActive.StatusReasonPtr()

					return leaseFromDB
				}, tt.retCreateErr)

			// Should initialize our lease usage step function
			sfnService := awsmocks.SFNAPI{}
			sfnService.
				On("StartExecution", mock.MatchedBy(func(input *sfn.StartExecutionInput) bool {
					// Check the state machine ARN, matches our configured value
					// (from env vars)
					assert.Equal(t, "mock-step-function-arn", *input.StateMachineArn)

					// Deserialize the JSON input
					var inputData map[string]interface{}
					err := json.Unmarshal([]byte(*input.Input), &inputData)
					assert.Nil(t, err)

					// Check the values of the JSON input,
					// should be equal to our lease object
					assert.Equal(t, *leaseFromDB.ID, inputData["id"].(string))
					assert.Equal(t, *leaseFromDB.AccountID, inputData["accountId"].(string))
					assert.Equal(t, *leaseFromDB.PrincipalID, inputData["principalId"].(string))
					assert.Equal(t, "Active", inputData["leaseStatus"].(string))

					return true
				})).
				Return(&sfn.StartExecutionOutput{}, nil)

			svcBldr.Config.
				WithService(&accountSvc).
				WithService(&leaseSvc).
				WithService(&userDetailSvc).
				WithService(&sfnService).
				WithEnv("PrincipalBudgetPeriod", "PRINCIPAL_BUDGET_PERIOD", "Weekly")
			_, err := svcBldr.Build()

			Settings.UsageStepFunctionArn = "mock-step-function-arn"

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			requestBody, err := json.Marshal(tt.requestBody)
			assert.Nil(t, err)
			request := events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       string(requestBody),
			}
			resp, err := Handler(context.TODO(), request)

			assert.Nil(t, err)

			assert.Equal(t, tt.expectedResponseCode, resp.StatusCode)

			var respJSON map[string]interface{}
			err = json.Unmarshal([]byte(resp.Body), &respJSON)
			require.Nil(t, err)
			assert.Equal(t, tt.expectedResponseBody, respJSON, "response JSON body")
			assert.Equal(t, map[string][]string{
				"Access-Control-Allow-Origin": {"*"},
				"Content-Type":                {"application/json"},
			}, resp.MultiValueHeaders)

			if tt.expectedResponseCode < 400 {
				accountSvc.AssertExpectations(t)
				leaseSvc.AssertExpectations(t)
				userDetailSvc.AssertExpectations(t)
				sfnService.AssertExpectations(t)
			}
		})
	}

}
