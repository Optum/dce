package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	util "github.com/Optum/dce/tests/testutils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/mock"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	commonMock "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	mockDB "github.com/Optum/dce/pkg/db/mocks"
	mockUsage "github.com/Optum/dce/pkg/usage/mocks"
	"github.com/aws/aws-lambda-go/events"
)

func TestCreateController_Call(t *testing.T) {

	t.Run("should create leases", func(t *testing.T) {
		type (
			fields struct {
				Dao                   db.DBer
				SNS                   common.Notificationer
				LeaseTopicARN         *string
				PrincipalBudgetAmount *float64
				PrincipalBudgetPeriod *string
				MaxLeaseBudgetAmount  *float64
				MaxLeasePeriod        *int64
			}
		)
		type args struct {
			ctx context.Context
			req *events.APIGatewayProxyRequest
		}

		leaseTopicARN := "some-topic-arn"
		messageID := "message123456789"

		amount := 1000.00
		period := "WEEKLY"
		leasePeriod := int64(704800)

		principalBudgetAmount = amount
		principalBudgetPeriod = period
		maxLeaseBudgetAmount = amount
		maxLeasePeriod = leasePeriod

		dbMock := stubDb()
		snsMock := &commonMock.Notificationer{}
		usageMock := &mockUsage.DBer{}
		sevenDaysOut := time.Now().AddDate(0, 0, 7).Unix()

		// Should create/update the lease record
		util.ReplaceMock(&dbMock.Mock, "UpsertLease", mock.MatchedBy(func(lease db.Lease) bool {
			assert.Equal(t, "123456789012", lease.AccountID)
			assert.Equal(t, "jdoe123", lease.PrincipalID)
			assert.IsType(t, "string", lease.ID)
			assert.Equal(t, db.Active, lease.LeaseStatus)
			assert.Equal(t, db.LeaseActive, lease.LeaseStatusReason)
			assert.Equal(t, float64(50), lease.BudgetAmount)
			assert.Equal(t, "USD", lease.BudgetCurrency)
			assert.Equal(t, []string{"user3@example.com", "user2@example.com"}, lease.BudgetNotificationEmails)
			assert.Equal(t, sevenDaysOut, lease.ExpiresOn)

			assert.InDelta(t, time.Now().Unix(), lease.CreatedOn, 2)
			assert.Equal(t, lease.CreatedOn, lease.LastModifiedOn)
			assert.Equal(t, lease.CreatedOn, lease.LeaseStatusModifiedOn)

			return true
		})).
			Return(&db.Lease{}, nil)

		// Should publish SNS message
		snsMock.On("PublishMessage", &leaseTopicARN, mock.Anything, true).Return(&messageID, nil)
		usageMock.On("GetUsageByPrincipal", mock.Anything, mock.Anything).Return(nil, nil)

		testFields := &fields{
			Dao:                   dbMock,
			SNS:                   snsMock,
			LeaseTopicARN:         &leaseTopicARN,
			PrincipalBudgetAmount: &principalBudgetAmount,
			PrincipalBudgetPeriod: &principalBudgetPeriod,
			MaxLeaseBudgetAmount:  &maxLeaseBudgetAmount,
			MaxLeasePeriod:        &maxLeasePeriod,
		}

		successResponse := createSuccessCreateResponse()
		badRequestResponse := response.CreateMultiValueHeaderAPIErrorResponse(http.StatusBadRequest, "RequestValidationError", "invalid request parameters")
		pastRequestResponse := response.RequestValidationError("Requested lease has a desired expiry date less than today: 1570627876")
		invalidBudgetRequestResponse := response.RequestValidationError("Requested lease has a budget amount of 5000.000000, which is greater than max lease budget amount of 1000.000000")
		invalidBudgetPeriodRequestResponse := response.RequestValidationError("Requested lease has a budget expires on of 1577745392, which is greater than max lease period of 1573176192")

		successArgs := &args{ctx: context.Background(), req: createSuccessfulCreateRequest()}
		pastArgs := &args{ctx: context.Background(), req: createPastCreateRequest()}
		invalidBudgetArgs := &args{ctx: context.Background(), req: invalidBudgetAmountCreateRequest()}
		invalidBudgetPeriodArgs := &args{ctx: context.Background(), req: invalidBudgetPeriodCreateRequest()}
		badArgs := &args{ctx: context.Background(), req: createBadCreateRequest()}

		tests := []struct {
			name    string
			fields  fields
			args    args
			want    events.APIGatewayProxyResponse
			wantErr bool
		}{
			{
				name:    "Bad request.",
				fields:  *testFields,
				args:    *badArgs,
				want:    badRequestResponse,
				wantErr: false,
			},
			{
				name:    "Past request.",
				fields:  *testFields,
				args:    *pastArgs,
				want:    pastRequestResponse,
				wantErr: false,
			},
			{
				name:    "Invalid budget amount request.",
				fields:  *testFields,
				args:    *invalidBudgetArgs,
				want:    invalidBudgetRequestResponse,
				wantErr: false,
			},
			{
				name:    "Invalid budget period request.",
				fields:  *testFields,
				args:    *invalidBudgetPeriodArgs,
				want:    invalidBudgetPeriodRequestResponse,
				wantErr: false,
			},
			{
				name: "Successful create.",
				fields: fields{
					Dao:                   dbMock,
					SNS:                   snsMock,
					LeaseTopicARN:         &leaseTopicARN,
					PrincipalBudgetAmount: aws.Float64(9999999999),
					PrincipalBudgetPeriod: &principalBudgetPeriod,
					MaxLeaseBudgetAmount:  aws.Float64(9999999999),
					MaxLeasePeriod:        aws.Int64(600000000),
				},
				args:    *successArgs,
				want:    *successResponse,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dao = tt.fields.Dao
				snsSvc = tt.fields.SNS
				leaseAddedTopicARN = *tt.fields.LeaseTopicARN
				principalBudgetAmount = *tt.fields.PrincipalBudgetAmount
				principalBudgetPeriod = *tt.fields.PrincipalBudgetPeriod
				maxLeaseBudgetAmount = *tt.fields.MaxLeaseBudgetAmount
				maxLeasePeriod = *tt.fields.MaxLeasePeriod
				got, err := Handler(tt.args.ctx, *tt.args.req)
				if (err != nil) != tt.wantErr {
					t.Errorf("CreateController.Call() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {

					//comparing first 50 characters of error message for invalid budget period error
					if strings.HasPrefix(tt.want.Body, got.Body[:30]) {
						return
					}
					t.Errorf("CreateController.Call() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("should fail if the principal already has a lease", func(t *testing.T) {
		// Mock active lease for the principal
		dbMock := stubDb()
		util.ReplaceMock(&dbMock.Mock, "FindLeasesByPrincipal", "jdoe123").
			Return([]*db.Lease{{
				AccountID:   "123456789012",
				LeaseStatus: db.Active,
			}}, nil)

		// Create the controller
		dao = dbMock

		// Call the controller
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "jdoe123",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
			"expiresOn":      time.Now().AddDate(0, 0, 7).Unix(),
		}))
		require.Nil(t, err)
		// Check HTTP error response
		require.Equal(t,
			response.CreateMultiValueHeaderAPIErrorResponse(http.StatusConflict, "ClientError", "Principal already has an active lease for account 123456789012"),
			res,
		)
	})

	t.Run("should mark the account.Status=Leased", func(t *testing.T) {
		// Setup the controller
		dbMock := stubDb()
		dao = dbMock

		// Return a ready account
		util.ReplaceMock(&dbMock.Mock, "GetReadyAccount").
			Return(&db.Account{ID: "123456789012"}, nil)

		// Should set account.Status=Leased
		util.ReplaceMock(&dbMock.Mock, "TransitionAccountStatus",
			"123456789012", db.Ready, db.Leased,
		).Return(&db.Account{}, nil)

		// Call the controller
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "jdoe123",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
			"expiresOn":      time.Now().AddDate(0, 0, 7).Unix(),
		}))
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode)

		dbMock.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
	})

	t.Run("should set default expiresOn", func(t *testing.T) {
		dbMock := stubDb()

		// Should set expiresOn to 7 days from now
		util.ReplaceMock(&dbMock.Mock, "UpsertLease", mock.MatchedBy(func(lease db.Lease) bool {
			assert.InDelta(t, time.Now().AddDate(0, 0, 7).UTC().Unix(), lease.ExpiresOn, 2)

			return true
		})).Return(func(lease db.Lease) *db.Lease {
			return &lease
		}, nil)

		dao = dbMock

		// Call the controller
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "jdoe123",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
		}))
		require.Nil(t, err)

		require.Equal(t, 201, res.StatusCode)
		resJSON := unmarshal(t, res.Body)
		require.InDelta(t, time.Now().AddDate(0, 0, 7).UTC().Unix(), resJSON["expiresOn"], 2)
	})

	t.Run("should create lease with metadata", func(t *testing.T) {
		// Setup the controller
		dbMock := stubDb()
		dao = dbMock

		// Should put lease metadata to DB
		util.ReplaceMock(&dbMock.Mock, "UpsertLease",
			mock.MatchedBy(func(lease db.Lease) bool {
				assert.Equal(t, map[string]interface{}{
					"foo": "bar",
					"faz": "baz",
				}, lease.Metadata)
				return true
			}),
		).Return(func(lease db.Lease) *db.Lease {
			return &lease
		}, nil)

		// Call the controller with some metadata
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "pid",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
			"metadata": map[string]interface{}{
				"foo": "bar",
				"faz": "baz",
			},
		}))
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode)

		// Check that controller responded with the metadata
		resJSON := unmarshal(t, res.Body)
		require.Contains(t, resJSON, "metadata")
		require.Equal(t, map[string]interface{}{
			"foo": "bar",
			"faz": "baz",
		}, resJSON["metadata"])

		// Verify our DB call
		dbMock.AssertExpectations(t)
	})

	t.Run("should allow complex types in metadata", func(t *testing.T) {
		// Setup the controller
		// Setup the controller
		dbMock := stubDb()
		dao = dbMock

		// Should put lease metadata to DB
		util.ReplaceMock(&dbMock.Mock, "UpsertLease",
			mock.MatchedBy(func(lease db.Lease) bool {
				assert.Equal(t, map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": map[string]interface{}{
							"faz":   "baz",
							"int":   float64(5),
							"float": 1.2345,
							"bool":  true,
							"nil":   nil,
						},
					},
				}, lease.Metadata)
				return true
			}),
		).Return(func(lease db.Lease) *db.Lease {
			return &lease
		}, nil)

		// Call the controller with some metadata
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "pid",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
			"metadata": map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": map[string]interface{}{
						"faz":   "baz",
						"int":   5,
						"float": 1.2345,
						"bool":  true,
						"nil":   nil,
					},
				},
			},
		}))
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode)

		// Check that controller responded with the metadata
		resJSON := unmarshal(t, res.Body)
		require.Contains(t, resJSON, "metadata")
		require.Equal(t, map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": map[string]interface{}{
					"faz":   "baz",
					"int":   float64(5),
					"float": 1.2345,
					"bool":  true,
					"nil":   nil,
				},
			},
		}, resJSON["metadata"])

		// Verify our DB call
		dbMock.AssertExpectations(t)
	})

	t.Run("should default to an empty metadata object", func(t *testing.T) {
		// Setup the controller
		// Setup the controller
		dbMock := stubDb()
		dao = dbMock

		// Should put lease metadata to DB
		util.ReplaceMock(&dbMock.Mock, "UpsertLease",
			mock.MatchedBy(func(lease db.Lease) bool {
				// Should save empty metadata to DB
				assert.Equal(t, map[string]interface{}{}, lease.Metadata)
				return true
			}),
		).Return(func(lease db.Lease) *db.Lease {
			return &lease
		}, nil)

		// Call the controller with no metadata
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "pid",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
		}))
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode)

		// Check that controller responded with empty metadata
		resJSON := unmarshal(t, res.Body)
		require.Contains(t, resJSON, "metadata")
		require.Equal(t, map[string]interface{}{}, resJSON["metadata"])

		// Verify our DB call
		dbMock.AssertExpectations(t)
	})

	t.Run("should not allow non-object types for metadata", func(t *testing.T) {

		// Metadata must be a JSON object
		invalidMetadatas := []interface{}{
			"string",
			14.5,
			true,
		}

		for _, metadata := range invalidMetadatas {
			// Call the controller with invalid metadata values
			res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
				"principalId":    "pid",
				"budgetAmount":   100,
				"budgetCurrency": "USD",
				"metadata":       metadata,
			}))
			require.Nil(t, err)

			// Check HTTP error response
			require.Equalf(t,
				response.RequestValidationError("invalid request parameters"),
				res,
				"should fail for metadata: %s", metadata,
			)
		}
	})

	t.Run("should deactivate the lease if the account status update fails", func(t *testing.T) {
		// Setup the controller
		dbMock := stubDb()
		dao = dbMock

		// Mock account status update to fail
		util.ReplaceMock(&dbMock.Mock, "TransitionAccountStatus",
			mock.Anything, mock.Anything, mock.Anything,
		).Return(nil, errors.New("test error"))

		// Should deactivate the lease
		util.ReplaceMock(&dbMock.Mock, "TransitionLeaseStatus",
			"123456789012", "jdoe123",
			db.Active, db.Inactive, db.LeaseRolledBack,
		).Return(&db.Lease{}, nil)

		// Call the controller
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "jdoe123",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
		}))
		require.Nil(t, err)

		// Should return a 500 error
		require.Equal(t, response.CreateMultiValueHeaderAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"), res)

		// Should have deactivated lease
		dbMock.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
	})

	t.Run("should deactivate the lease and update the account status if the SNS publish fails", func(t *testing.T) {
		// Mock SNS to fail
		snsMock := &commonMock.Notificationer{}
		snsMock.On("PublishMessage", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("test error"))

		dbMock := stubDb()
		snsSvc = snsMock
		dao = dbMock

		// Should deactivate the lease
		util.ReplaceMock(&dbMock.Mock, "TransitionLeaseStatus",
			"123456789012", "jdoe123",
			db.Active, db.Inactive, db.LeaseRolledBack,
		).Return(&db.Lease{}, nil)

		// Should mark Account.Status=Ready
		dbMock.On("TransitionAccountStatus",
			"123456789012", db.Leased, db.Ready,
		).Return(&db.Account{}, nil)

		// Call the controller
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "jdoe123",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
		}))
		require.Nil(t, err)

		// Should return a 500 error
		require.Equal(t, response.CreateMultiValueHeaderAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"), res)

		// Should have deactivated lease
		dbMock.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
		dbMock.AssertNumberOfCalls(t, "TransitionAccountStatus", 2)
	})

	// ...otherwise, we'd end up with a Lease=Active, Account=Ready state.
	t.Run("should not rollback accounts status on SNS rollback, if the lease deactivation fails", func(t *testing.T) {
		// Mock SNS to fail
		snsMock := &commonMock.Notificationer{}
		snsMock.On("PublishMessage", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("test error"))

		dbMock := stubDb()
		snsSvc = snsMock
		dao = dbMock

		// Should deactivate the lease (FAILS)
		util.ReplaceMock(&dbMock.Mock, "TransitionLeaseStatus",
			"123456789012", "jdoe123",
			db.Active, db.Inactive, db.LeaseRolledBack,
		).Return(nil, errors.New("test error"))

		// Call the controller
		res, err := Handler(context.TODO(), *apiGatewayRequest(t, map[string]interface{}{
			"principalId":    "jdoe123",
			"budgetAmount":   100,
			"budgetCurrency": "USD",
		}))
		require.Nil(t, err)

		// Should return a 500 error
		require.Equal(t, response.CreateMultiValueHeaderAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"), res)

		// Should have deactivated lease
		dbMock.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
		// Should only call this once, for the original Account Status=Leased
		dbMock.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
	})

}

func createSuccessfulCreateRequest() *events.APIGatewayProxyRequest {
	sevenDaysOut := time.Now().AddDate(0, 0, 7).Unix()
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "jdoe123",
		BudgetAmount:             50,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                sevenDaysOut,
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

func apiGatewayRequest(t *testing.T, jsonObj map[string]interface{}) *events.APIGatewayProxyRequest {
	reqBody, err := json.Marshal(jsonObj)
	require.Nilf(t, err, "failed to marshal JSON: %v", jsonObj)

	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/leases",
		Body:       string(reqBody),
	}
}

func createBadCreateRequest() *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/leases",
	}
}

func createPastCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "jdoe123",
		BudgetAmount:             50,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                1570627876,
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

func createSuccessCreateResponse() *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		StatusCode: 201,
		Body:       "{\"accountId\":\"\",\"principalId\":\"\",\"id\":\"\",\"leaseStatus\":\"\",\"leaseStatusReason\":\"\",\"createdOn\":0,\"lastModifiedOn\":0,\"budgetAmount\":0,\"budgetCurrency\":\"\",\"budgetNotificationEmails\":null,\"leaseStatusModifiedOn\":0,\"expiresOn\":0,\"metadata\":{}}",
	}
}

func unmarshal(t *testing.T, jsonStr string) map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	require.Nil(t, err,
		fmt.Sprintf("Failed to unmarshal JSON: %s; %s", jsonStr, err),
	)

	return data
}

func invalidBudgetAmountCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "123456",
		BudgetAmount:             5000,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                time.Now().AddDate(0, 0, 5).Unix(),
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

func invalidBudgetPeriodCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "123456",
		BudgetAmount:             50,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                time.Now().AddDate(0, 2, 0).Unix(),
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

// stubDb creates a mock DBer,
// with stub/no-op mocks for each method used by the create lease controller
func stubDb() *mockDB.DBer {
	dbMock := &mockDB.DBer{}

	// Return a ready account
	dbMock.On("GetReadyAccount").
		Return(&db.Account{ID: "123456789012"}, nil)

	// Requesting principal has no leases in the DB
	dbMock.On("FindLeasesByPrincipal", mock.Anything).Return(nil, nil)

	// Should upsert the lease DB record
	dbMock.On("UpsertLease", mock.Anything).
		// Return the same lease object that was passed to this method
		Return(func(lease db.Lease) *db.Lease {
			return &lease
		}, nil)

	// Transitions account to status=Leased
	dbMock.On("TransitionAccountStatus", mock.Anything, mock.Anything, mock.Anything).
		Return(&db.Account{}, nil)

	return dbMock
}
