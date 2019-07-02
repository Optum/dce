package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	commonMocks "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbMocks "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {

	t.Run("should return a RedboxAccount object", func(t *testing.T) {
		// Send request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:*:*:*",
		})
		controller := newCreateAccountController()
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		// Unmarshal the response JSON into an account object
		resJSON := unmarshal(t, res.Body)

		require.Equal(t, "1234567890", resJSON["id"])
		require.Equal(t, "arn:*:*:*", resJSON["adminRoleArn"])
		require.Equal(t, "NotReady", resJSON["accountStatus"])
		require.True(t, resJSON["lastModifiedOn"].(float64) > 1561518000)
		require.True(t, resJSON["createdOn"].(float64) > 1561518000)
	})

	t.Run("should fail if adminRoleArn is missing", func(t *testing.T) {
		// Send request, missing AdminRoleArn
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "",
		})
		controller := newCreateAccountController()
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		// Check the error response
		require.Equal(t,
			response.RequestValidationError("missing required field \"adminRoleArn\""),
			res,
			"should return a validation error",
		)
	})

	t.Run("should fail if accountID is missing", func(t *testing.T) {
		// Send request, missing AdminRoleArn
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "",
			AdminRoleArn: "arn:mock",
		})
		controller := newCreateAccountController()
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		// Check the error response
		require.Equal(t,
			response.RequestValidationError("missing required field \"id\""),
			res,
			"should return a validation error",
		)
	})

	t.Run("should fail if the adminRoleArn is not assumable", func(t *testing.T) {
		// Configure a controller with a mock token service
		controller := newCreateAccountController()
		tokenService := &commonMocks.TokenService{}
		controller.TokenService = tokenService

		// Should fail to assume role
		tokenService.On("AssumeRole",
			mock.MatchedBy(func(input *sts.AssumeRoleInput) bool {
				require.Equal(t, "arn:iam:adminRole", *input.RoleArn)
				require.Equal(t, "RedboxMasterAssumeRoleVerification", *input.RoleSessionName)

				return true
			}),
		).Return(nil, errors.New("mock error, failed to assume role"))
		defer tokenService.AssertExpectations(t)

		// Call the controller
		res, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createAccountRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:iam:adminRole",
			}),
		)
		require.Nil(t, err)
		require.Equal(t,
			response.RequestValidationError("Unable to create Account: adminRole is not assumable by the Redbox master account"),
			res,
		)
	})

	t.Run("should add the account to the RedboxAccounts DB Table, as NotReady", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateAccountController()
		controller.Dao = mockDb

		// Mock the DB, so that the account doesn't already exist
		mockDb.On("GetAccount", "1234567890").
			Return(nil, nil)

		// Mock the DB method to create the Account
		mockDb.On("PutAccount",
			mock.MatchedBy(func(account db.RedboxAccount) bool {
				require.Equal(t, "1234567890", account.ID)
				require.Equal(t, "arn:mock", account.AdminRoleArn)
				return true
			}),
		).Return(nil)
		defer mockDb.AssertExpectations(t)

		// Send an API request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode)
	})

	t.Run("should return a 409 if the account already exists", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateAccountController()
		controller.Dao = mockDb

		// Mock the DB, so that the account already exist
		mockDb.On("GetAccount", "1234567890").
			Return(&db.RedboxAccount{}, nil)

		// Send an API request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		require.Equal(t, response.AlreadyExistsError(), res)
	})

	t.Run("should handle DB.GetAccount response errors as 500s", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateAccountController()
		controller.Dao = mockDb

		// Mock the DB to return an error
		mockDb.On("GetAccount", "1234567890").
			Return(nil, errors.New("mock error"))

		// Send an API request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		require.Equal(t, response.ServerError(), res)
	})

	t.Run("should handle DB.PutAccount response errors as 500s", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateAccountController()
		controller.Dao = mockDb

		// Account doesn't already exist
		mockDb.On("GetAccount", "1234567890").
			Return(nil, nil)

		// Mock the db to return an error
		mockDb.On("PutAccount", mock.Anything).
			Return(errors.New("mock error"))

		// Send an API request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		require.Equal(t, response.ServerError(), res)
	})

	t.Run("should add the account to the reset Queue", func(t *testing.T) {
		// Configure the controller, with a mock SQS
		mockQueue := &commonMocks.Queue{}
		controller := newCreateAccountController()
		controller.Queue = mockQueue
		controller.ResetQueueURL = "mock.queue.url"

		// Should add account to Queue
		mockQueue.On("SendMessage",
			aws.String("mock.queue.url"),
			aws.String("1234567890"),
		).Return(nil)
		defer mockQueue.AssertExpectations(t)

		// Send request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode, res.Body)
	})

	t.Run("should return a 500, if SQS fails", func(t *testing.T) {
		// Configure the controller, with a mock SQS
		mockQueue := &commonMocks.Queue{}
		mockDB := dbStub()
		controller := newCreateAccountController()
		controller.Dao = mockDB
		controller.Queue = mockQueue
		controller.ResetQueueURL = "mock.queue.url"

		// Should fail to add account to Queue
		mockQueue.On("SendMessage",
			aws.String("mock.queue.url"),
			aws.String("1234567890"),
		).Return(errors.New("mock error"))
		defer mockQueue.AssertExpectations(t)

		// Send request
		req := createAccountAPIRequest(t, createAccountRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		// Should return an InternalServerError
		require.Equal(t, response.ServerError(), res)

		// Account should still be saved to DB, in `NotReady`
		// state (to be reset later)
		mockDB.AssertExpectations(t)
	})

	t.Run("should publish an SNS message, with the account info", func(t *testing.T) {
		// Configure the controller with mock SNS
		mockSNS := &commonMocks.Notificationer{}
		controller := newCreateAccountController()
		controller.SNS = mockSNS
		controller.AccountCreatedTopicArn = "mock-account-created-topic"

		// Expect to publish the account to the SNS topic
		mockSNS.On("PublishMessage",
			mock.MatchedBy(func(arn *string) bool {
				return *arn == "mock-account-created-topic"
			}),
			mock.MatchedBy(func(message *string) bool {
				// Parse the message JSON
				messageObj := unmarshal(t, *message)
				// `default` and `body` and JSON embedded within the message JSON
				msgDefault := unmarshal(t, messageObj["default"].(string))
				msgBody := unmarshal(t, messageObj["Body"].(string))

				require.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

				// Check that we're sending the account object
				require.Equal(t, "1234567890", msgBody["id"])
				require.Equal(t, "arn:mockAdmin", msgBody["adminRoleArn"])
				require.Equal(t, "NotReady", msgBody["accountStatus"])
				require.IsType(t, 0.0, msgBody["lastModifiedOn"])
				require.IsType(t, 0.0, msgBody["createdOn"])

				return true
			}),
			true,
		).Return(aws.String("mock message"), nil)
		defer mockSNS.AssertExpectations(t)

		// Call the controller with the account
		res, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createAccountRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mockAdmin",
			}),
		)
		require.Nil(t, err)
		require.Equal(t, res.StatusCode, 201)
	})

	t.Run("should return a 500, if the SNS publish fails", func(t *testing.T) {
		// Configure the controller with mock SNS
		mockSNS := &commonMocks.Notificationer{}
		controller := newCreateAccountController()
		controller.SNS = mockSNS

		// Mock SNS publish to fail
		mockSNS.On("PublishMessage", mock.Anything, mock.Anything, mock.Anything).
			Return(aws.String(""), errors.New("mock SNS error"))

		// Call the controller with the account
		res, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createAccountRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mockAdmin",
			}),
		)
		require.Nil(t, err)

		// Should return a ServerError
		require.Equal(t, response.ServerError(), res)
	})

}

// dbStub creates a mock DBer instance,
// preconfigured to follow happy-path behavior
func dbStub() *dbMocks.DBer {
	mockDb := &dbMocks.DBer{}
	// Mock the DB, so that the account doesn't already exist
	mockDb.On("GetAccount", mock.Anything).
		Return(nil, nil)
	mockDb.On("PutAccount", mock.Anything).Return(nil)

	return mockDb
}

func queueStub() *commonMocks.Queue {
	mockQueue := &commonMocks.Queue{}
	mockQueue.On("SendMessage", mock.Anything, mock.Anything).
		Return(nil)

	return mockQueue
}

func snsStub() *commonMocks.Notificationer {
	mockSNS := &commonMocks.Notificationer{}
	mockSNS.On("PublishMessage", mock.Anything, mock.Anything, mock.Anything).
		Return(aws.String("mock-message-id"), nil)

	return mockSNS
}

func tokenServiceStub() common.TokenService {
	tokenServiceMock := &commonMocks.TokenService{}
	tokenServiceMock.On("AssumeRole", mock.Anything).
		Return(nil, nil)

	return tokenServiceMock
}

func createAccountAPIRequest(t *testing.T, req createAccountRequest) *events.APIGatewayProxyRequest {
	requestBody, err := json.Marshal(&req)
	require.Nil(t, err)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBody),
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

func newCreateAccountController() createAccountController {
	return createAccountController{
		Dao:          dbStub(),
		Queue:        queueStub(),
		SNS:          snsStub(),
		TokenService: tokenServiceStub(),
	}
}
