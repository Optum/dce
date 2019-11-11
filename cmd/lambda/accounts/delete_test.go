package main

import (
	"context"
	"errors"
	"net/http"
	"testing"

	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/rolemanager"
	mocks2 "github.com/Optum/dce/pkg/rolemanager/mocks"
	roleManagerMocks "github.com/Optum/dce/pkg/rolemanager/mocks"

	"github.com/Optum/dce/pkg/api/response"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/common"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteController_Call(t *testing.T) {
	expectedAccount := db.Account{
		ID:           "123456789012",
		AdminRoleArn: "arn:admin-role",
	}
	t.Run("When there are no errors", func(t *testing.T) {

		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "123456789012").Return(&expectedAccount, nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789012"}

		mockAwsSession := &awsMocks.AwsSession{}
		mockAwsSession.On("ClientConfig", mock.Anything).Return(client.Config{
			Config: &aws.Config{},
		})

		mockTokenService := commonMocks.TokenService{}
		mockTokenService.On("NewSession", mock.Anything, "arn:admin-role").Return(mockAwsSession, nil)

		roleManager := roleManagerMocks.RoleManager{}
		roleManager.On("SetIAMClient", mock.Anything)
		roleManager.On("DestroyRoleWithPolicy", mock.Anything).Return(nil, nil)

		mockSns := commonMocks.Notificationer{}
		mockSns.On("PublishMessage", mock.Anything, mock.Anything, true).Return(nil, nil)

		mockQueue := commonMocks.Queue{}
		mockQueue.On("SendMessage", mock.Anything, mock.Anything).Return(nil)

		// AWSSession = &session
		Dao = &mockDb
		TokenSvc = &mockTokenService
		RoleManager = &roleManager
		SnsSvc = &mockSns
		Queue = &mockQueue

		response, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusNoContent, response.StatusCode)
	})

	t.Run("When the account is not found", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "123456789012").Return(nil, &db.AccountNotFoundError{})
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789012"}

		Dao = &mockDb

		response, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("When the account is leased", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "123456789012").Return(&expectedAccount, &db.AccountLeasedError{})
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789012"}

		mockAwsSession := &awsMocks.AwsSession{}
		mockAwsSession.On("ClientConfig", mock.Anything).Return(client.Config{
			Config: &aws.Config{},
		})

		mockTokenService := commonMocks.TokenService{}
		mockTokenService.On("NewSession", mock.Anything, "arn:admin-role").Return(mockAwsSession, nil)

		Dao = &mockDb
		TokenSvc = &mockTokenService
		Queue = queueStub()
		SnsSvc = snsStub()

		RoleManager = roleManagerStub()
		response, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusConflict, response.StatusCode)
	})

	t.Run("When handling any other error", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "123456789012").Return(&expectedAccount, errors.New("Test"))
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789012"}
		Dao = &mockDb
		response, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	})

	t.Run("should destroy the principal IAM Role and Policy", func(t *testing.T) {

		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "123456789012").Return(&expectedAccount, nil)

		Dao = &mockDb

		// Mock the role manager
		roleManager := &mocks2.RoleManager{}
		RoleManager = roleManager

		// Mock RoleManager.DestroyRoleWithPolicy()
		roleManager.On("DestroyRoleWithPolicy", &rolemanager.DestroyRoleWithPolicyInput{
			RoleName:  "DCEPrincipal",
			PolicyArn: "arn:aws:iam::123456789012:policy/DCEPrincipalDefaultPolicy",
		}).Return(&rolemanager.DestroyRoleWithPolicyOutput{}, nil)

		// Should set the IAM role (using the assumed account creds)
		roleManager.On("SetIAMClient", mock.Anything)

		// Call the controller
		res, err := Handler(context.TODO(), mockDeleteRequest("123456789012"))
		assert.Nil(t, err)
		assert.Equal(t, MockAPIResponse(http.StatusNoContent, ""), res)

		roleManager.AssertExpectations(t)
	})

	t.Run("should return 204, even if the principal role cannot be deleted", func(t *testing.T) {

		// Mock the role manager
		roleManager := &mocks2.RoleManager{}
		RoleManager = roleManager

		// Mock RoleManager.DestroyRoleWithPolicy() to return an error
		roleManager.On("DestroyRoleWithPolicy", &rolemanager.DestroyRoleWithPolicyInput{
			RoleName:  "DCEPrincipal",
			PolicyArn: "arn:aws:iam::123456789012:policy/DCEPrincipalDefaultPolicy",
		}).Return(nil, &errors2.MultiError{})

		// Should set the IAM role (using the assumed account creds)
		roleManager.On("SetIAMClient", mock.Anything)

		// Call the controller
		res, err := Handler(context.TODO(), mockDeleteRequest("123456789012"))
		assert.Nil(t, err)
		assert.Equal(t, MockAPIResponse(http.StatusNoContent, ""), res)

		roleManager.AssertExpectations(t)
	})

	t.Run("Sending the accountID to the queue", func(t *testing.T) {
		expectedResetQueueURL := "mock.queue.url"
		expectedAccountID := "123456789012"
		stub := &commonMocks.Queue{}
		stub.On("SendMessage", &expectedResetQueueURL, &expectedAccountID).Return(nil)

		Queue = stub

		sendToResetQueue(expectedAccountID)
		stub.AssertCalled(t, "SendMessage", &expectedResetQueueURL, &expectedAccountID)
	})

	t.Run("Sending the send SNS", func(t *testing.T) {
		expectedArn := "test:arn"
		expectedReturned := "return"
		serializedAccount := response.AccountResponse(expectedAccount)
		serializedMessage, err := common.PrepareSNSMessageJSON(serializedAccount)
		assert.Nil(t, err)

		stub := &commonMocks.Notificationer{}
		stub.On("PublishMessage", &expectedArn, &serializedMessage, true).Return(&expectedReturned, nil)

		SnsSvc = stub

		sendSNS(&expectedAccount)
		stub.AssertCalled(t, "PublishMessage", &expectedArn, &serializedMessage, true)
	})
}

func mockDeleteRequest(accountID string) events.APIGatewayProxyRequest {
	return events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodDelete,
		Path:       "/accounts/" + accountID,
	}
}
