package main

import (
	"context"
	"errors"
	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/rolemanager"
	mocks2 "github.com/Optum/dce/pkg/rolemanager/mocks"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func TestDeleteController_Call(t *testing.T) {
	expectedAccount := db.Account{
		ID: "1",
	}
	t.Run("When there are no errors", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(&expectedAccount, nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := newDeleteController()
		controller.Dao = &mockDb
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode)
	})

	t.Run("When the account is not found", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(nil, &db.AccountNotFoundError{})
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := newDeleteController()
		controller.Dao = &mockDb
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("When the account is leased", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(&expectedAccount, &db.AccountLeasedError{})
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := deleteController{
			Dao:                    &mockDb,
			Queue:                  queueStub(),
			SNS:                    snsStub(),
			AccountDeletedTopicArn: "test:arn",
			ResetQueueURL:          "www.test.com",
			RoleManager:            roleManagerStub(),
		}
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusConflict, response.StatusCode)
	})

	t.Run("When handling any other error", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(&expectedAccount, errors.New("Test"))
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := newDeleteController()
		controller.Dao = &mockDb
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusInternalServerError, response.StatusCode)
	})

	t.Run("should destroy the redbox principal IAM Role and Policy", func(t *testing.T) {
		controller := newDeleteController()

		// Mock the role manager
		roleManager := &mocks2.RoleManager{}
		controller.RoleManager = roleManager
		controller.PrincipalRoleName = "MockPrincipalRoleName"
		controller.PrincipalPolicyName = "MockPrincipalPolicyName"

		// Mock RoleManager.DestroyRoleWithPolicy()
		roleManager.On("DestroyRoleWithPolicy", &rolemanager.DestroyRoleWithPolicyInput{
			RoleName:  "MockPrincipalRoleName",
			PolicyArn: "arn:aws:iam::1234567890:policy/MockPrincipalPolicyName",
		}).Return(&rolemanager.DestroyRoleWithPolicyOutput{}, nil)

		// Should set the IAM role (using the assumed account creds)
		roleManager.On("SetIAMClient", mock.Anything)

		// Call the controller
		res, err := controller.Call(context.TODO(), mockDeleteRequest("1234567890"))
		require.Nil(t, err)
		require.Equal(t, response.CreateAPIResponse(http.StatusNoContent, ""), res)

		roleManager.AssertExpectations(t)
	})

	t.Run("should return 204, even if the redbox principal role cannot be deleted", func(t *testing.T) {
		controller := newDeleteController()

		// Mock the role manager
		roleManager := &mocks2.RoleManager{}
		controller.RoleManager = roleManager
		controller.PrincipalRoleName = "MockPrincipalRoleName"
		controller.PrincipalPolicyName = "MockPrincipalPolicyName"

		// Mock RoleManager.DestroyRoleWithPolicy() to return an error
		roleManager.On("DestroyRoleWithPolicy", &rolemanager.DestroyRoleWithPolicyInput{
			RoleName:  "MockPrincipalRoleName",
			PolicyArn: "arn:aws:iam::1234567890:policy/MockPrincipalPolicyName",
		}).Return(nil, &errors2.MultiError{})

		// Should set the IAM role (using the assumed account creds)
		roleManager.On("SetIAMClient", mock.Anything)

		// Call the controller
		res, err := controller.Call(context.TODO(), mockDeleteRequest("1234567890"))
		require.Nil(t, err)
		require.Equal(t, response.CreateAPIResponse(http.StatusNoContent, ""), res)

		roleManager.AssertExpectations(t)
	})

	t.Run("Sending the accountID to the queue", func(t *testing.T) {
		expectedResetQueueURL := "www.test.com"
		expectedAccountID := "12341234"
		stub := &commonMocks.Queue{}
		stub.On("SendMessage", &expectedResetQueueURL, &expectedAccountID).Return(nil)

		controller := deleteController{
			Queue:         stub,
			ResetQueueURL: expectedResetQueueURL,
		}

		controller.sendToResetQueue(expectedAccountID)
		stub.AssertCalled(t, "SendMessage", &expectedResetQueueURL, &expectedAccountID)
	})

	t.Run("Sending the send SNS", func(t *testing.T) {
		expectedArn := "test:arn"
		expectedReturned := "return"
		serializedAccount := response.AccountResponse(expectedAccount)
		serializedMessage, err := common.PrepareSNSMessageJSON(serializedAccount)
		require.Nil(t, err)

		stub := &commonMocks.Notificationer{}
		stub.On("PublishMessage", &expectedArn, &serializedMessage, true).Return(&expectedReturned, nil)

		controller := deleteController{
			SNS:                    stub,
			AccountDeletedTopicArn: expectedArn,
		}

		controller.sendSNS(&expectedAccount)
		stub.AssertCalled(t, "PublishMessage", &expectedArn, &serializedMessage, true)
	})
}

func newDeleteController() deleteController {
	return deleteController{
		Dao:                    dbStub(),
		Queue:                  queueStub(),
		SNS:                    snsStub(),
		AccountDeletedTopicArn: "test:arn",
		ResetQueueURL:          "www.test.com",
		RoleManager:            roleManagerStub(),
		TokenService:           tokenServiceStub(),
	}
}

func mockDeleteRequest(accountID string) *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodDelete,
		Path:       "/accounts/" + accountID,
	}
}
