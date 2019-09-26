package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	awsMocks "github.com/Optum/Dce/pkg/awsiface/mocks"
	"github.com/Optum/Dce/pkg/rolemanager"
	roleManagerMocks "github.com/Optum/Dce/pkg/rolemanager/mocks"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"

	"github.com/Optum/Dce/pkg/api/response"
	"github.com/Optum/Dce/pkg/common"
	commonMocks "github.com/Optum/Dce/pkg/common/mocks"
	"github.com/Optum/Dce/pkg/db"
	dbMocks "github.com/Optum/Dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {

	t.Run("should return a DceAccount object", func(t *testing.T) {
		// Send request
		req := createAccountAPIRequest(t, createRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:*:*:*",
		})
		controller := newCreateController()
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
		req := createAccountAPIRequest(t, createRequest{
			ID:           "1234567890",
			AdminRoleArn: "",
		})
		controller := newCreateController()
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
		req := createAccountAPIRequest(t, createRequest{
			ID:           "",
			AdminRoleArn: "arn:mock",
		})
		controller := newCreateController()
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
		controller := newCreateController()
		tokenService := &commonMocks.TokenService{}
		controller.TokenService = tokenService

		// Should fail to assume role
		tokenService.On("AssumeRole",
			mock.MatchedBy(func(input *sts.AssumeRoleInput) bool {
				assert.Equal(t, "arn:iam:adminRole", *input.RoleArn)
				assert.Equal(t, "DceMasterAssumeRoleVerification", *input.RoleSessionName)

				return true
			}),
		).Return(nil, errors.New("mock error, failed to assume role"))
		defer tokenService.AssertExpectations(t)

		// Call the controller
		res, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:iam:adminRole",
			}),
		)
		require.Nil(t, err)
		require.Equal(t,
			response.RequestValidationError("Unable to create Account: adminRole is not assumable by the Dce master account"),
			res,
		)
	})

	t.Run("should add the account to the DceAccounts DB Table, as NotReady", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateController()
		controller.PrincipalRoleName = "DcePrincipal"
		controller.Dao = mockDb

		// Mock the DB, so that the account doesn't already exist
		mockDb.On("GetAccount", "1234567890").
			Return(nil, nil)

		// Mock the DB method to create the Account
		mockDb.On("PutAccount",
			mock.MatchedBy(func(account db.DceAccount) bool {
				assert.Equal(t, "1234567890", account.ID)
				assert.Equal(t, "arn:mock", account.AdminRoleArn)
				assert.Equal(t, "arn:aws:iam::123456789012:role/DcePrincipal", account.PrincipalRoleArn)
				return true
			}),
		).Return(nil)
		defer mockDb.AssertExpectations(t)

		// Send an API request
		req := createAccountAPIRequest(t, createRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)
		require.Equal(t, 201, res.StatusCode)
	})

	t.Run("should return a 409 if the account already exists", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateController()
		controller.Dao = mockDb

		// Mock the DB, so that the account already exist
		mockDb.On("GetAccount", "1234567890").
			Return(&db.DceAccount{}, nil)

		// Send an API request
		req := createAccountAPIRequest(t, createRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		require.Equal(t, response.AlreadyExistsError(), res)
	})

	t.Run("should handle DB.GetAccount response errors as 500s", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateController()
		controller.Dao = mockDb

		// Mock the DB to return an error
		mockDb.On("GetAccount", "1234567890").
			Return(nil, errors.New("mock error"))

		// Send an API request
		req := createAccountAPIRequest(t, createRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		require.Equal(t, response.ServerError(), res)
	})

	t.Run("should handle DB.PutAccount response errors as 500s", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		controller := newCreateController()
		controller.Dao = mockDb

		// Account doesn't already exist
		mockDb.On("GetAccount", "1234567890").
			Return(nil, nil)

		// Mock the db to return an error
		mockDb.On("PutAccount", mock.Anything).
			Return(errors.New("mock error"))

		// Send an API request
		req := createAccountAPIRequest(t, createRequest{
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
		controller := newCreateController()
		controller.Queue = mockQueue
		controller.ResetQueueURL = "mock.queue.url"

		// Should add account to Queue
		mockQueue.On("SendMessage",
			aws.String("mock.queue.url"),
			aws.String("1234567890"),
		).Return(nil)
		defer mockQueue.AssertExpectations(t)

		// Send request
		req := createAccountAPIRequest(t, createRequest{
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
		controller := newCreateController()
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
		req := createAccountAPIRequest(t, createRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := controller.Call(context.TODO(), req)
		require.Nil(t, err)

		// Should return an InternalServerError
		require.Equal(t, response.ServerError(), res)

		// Account should still be saved to DB, in `NotReady`
		// state (to be reset later)
		mockDB.AssertCalled(t, "PutAccount", mock.Anything)
	})

	t.Run("should publish an SNS message, with the account info", func(t *testing.T) {
		// Configure the controller with mock SNS
		mockSNS := &commonMocks.Notificationer{}
		controller := newCreateController()
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

				assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

				// Check that we're sending the account object
				assert.Equal(t, "1234567890", msgBody["id"])
				assert.Equal(t, "arn:mockAdmin", msgBody["adminRoleArn"])
				assert.Equal(t, "NotReady", msgBody["accountStatus"])
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
			createAccountAPIRequest(t, createRequest{
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
		controller := newCreateController()
		controller.SNS = mockSNS

		// Mock SNS publish to fail
		mockSNS.On("PublishMessage", mock.Anything, mock.Anything, mock.Anything).
			Return(aws.String(""), errors.New("mock SNS error"))

		// Call the controller with the account
		res, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mockAdmin",
			}),
		)
		require.Nil(t, err)

		// Should return a ServerError
		require.Equal(t, response.ServerError(), res)
	})

	t.Run("should create a principal role and policy", func(t *testing.T) {
		// Create the controller
		controller := newCreateController()

		// Configure some parameters, to make sure these
		// get passed through to the IAM role
		controller.PrincipalMaxSessionDuration = 100
		controller.PrincipalRoleName = "DcePrincipal"
		controller.PrincipalPolicyName = "DcePrincipalDefaultPolicy"
		controller.PrincipalIAMDenyTags = []string{"Dce", "CantTouchThis"}
		controller.Tags = []*iam.Tag{{
			Key: aws.String("Foo"), Value: aws.String("Bar"),
		}}

		// Mock the TokenService (assumes role into the user account)
		tokenServiceMock := &commonMocks.TokenService{}
		controller.TokenService = tokenServiceMock

		// Mock Token Service, to assume adminRoleArn
		mockAdminRoleSession := &awsMocks.AwsSession{}
		mockAdminRoleSession.On("ClientConfig", mock.Anything).Return(client.Config{
			Config: &aws.Config{},
		})
		tokenServiceMock.On("NewSession", mock.Anything, "arn:mockAdmin").
			Return(mockAdminRoleSession, nil)
		tokenServiceMock.On("AssumeRole", mock.Anything).Return(nil, nil)

		// Mock the RoleManager (creates the IAM Role)
		roleManager := roleManagerMocks.RoleManager{}
		controller.RoleManager = &roleManager

		// RoleManager should use an IAM client,with the assumed role session
		roleManager.On("SetIAMClient", mock.Anything)

		// Setup expected AssumeRolePolicy
		expectedAssumeRolePolicy := strings.TrimSpace(`
		{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": "arn:aws:iam::1234567890:root"
					},
					"Action": "sts:AssumeRole",
					"Condition": {}
				}
			]
		}
		`)

		// Mock the RoleManager, to create an IAM Role for the Dce Principal
		roleManager.On("CreateRoleWithPolicy",
			mock.MatchedBy(func(input *rolemanager.CreateRoleWithPolicyInput) bool {
				// Verify the expected input
				assert.Equal(t, "DcePrincipal", input.RoleName)
				assert.Equal(t, "Role to be assumed by principal users of Dce", input.RoleDescription)
				assert.Equal(t, expectedAssumeRolePolicy, input.AssumeRolePolicyDocument)
				assert.Equal(t, int64(100), input.MaxSessionDuration)
				assert.Equal(t, "DcePrincipalDefaultPolicy", input.PolicyName)
				assert.Equal(t, []*iam.Tag{
					{Key: aws.String("Foo"), Value: aws.String("Bar")},
					{Key: aws.String("Name"), Value: aws.String("DcePrincipal")},
				}, input.Tags)
				assert.Equal(t, true, input.IgnoreAlreadyExistsErrors)
				assert.Equal(t, "", "")

				return true
			}),
		).Return(&rolemanager.CreateRoleWithPolicyOutput{}, nil)

		// Call the controller with the account
		_, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mockAdmin",
			}),
		)
		require.Nil(t, err)

		roleManager.AssertExpectations(t)
		tokenServiceMock.AssertExpectations(t)
	})

	t.Run("should return a 500 if creating the principal IAM role fails", func(t *testing.T) {
		// Create the controller
		controller := newCreateController()

		// Mock the RoleManager, to return an error on IAM Role Creation
		roleManager := roleManagerMocks.RoleManager{}
		controller.RoleManager = &roleManager
		roleManager.On("SetIAMClient", mock.Anything)
		roleManager.On("CreateRoleWithPolicy", mock.Anything).
			Return(nil, errors.New("mock error"))

		// Call the controller
		res, err := controller.Call(
			context.TODO(),
			createAccountAPIRequest(t, createRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mockAdmin",
			}),
		)
		require.Nil(t, err)

		// Should return a 500 Server Error
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
	mockDb.On("DeleteAccount", mock.Anything).
		Return(func(accountID string) *db.DceAccount {
			return &db.DceAccount{ID: accountID}
		}, nil)

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

	session := &awsMocks.AwsSession{}
	session.On("ClientConfig", mock.Anything).Return(client.Config{
		Config: &aws.Config{},
	})
	tokenServiceMock.On("NewSession", mock.Anything, mock.Anything).
		Return(session, nil)

	return tokenServiceMock
}

func StoragerMock() common.Storager {
	storagerMock := &commonMocks.Storager{}

	storagerMock.On("GetTemplateObject", mock.Anything, mock.Anything, mock.Anything).
		Return("", "", nil)

	return storagerMock
}

func roleManagerStub() *roleManagerMocks.RoleManager {
	roleManagerMock := &roleManagerMocks.RoleManager{}
	roleManagerMock.On("SetIAMClient", mock.Anything)
	roleManagerMock.On("CreateRoleWithPolicy", mock.Anything).
		Return(
			func(input *rolemanager.CreateRoleWithPolicyInput) *rolemanager.CreateRoleWithPolicyOutput {
				return &rolemanager.CreateRoleWithPolicyOutput{
					RoleName:   input.RoleName,
					RoleArn:    "arn:aws:iam::123456789012:role/" + input.RoleName,
					PolicyName: "DcePrincipalDefaultPolicy",
					PolicyArn:  "arn:aws:iam::1234567890:policy/DcePrincipalDefaultPolicy",
				}
			}, nil,
		)
	roleManagerMock.On("DestroyRoleWithPolicy", mock.Anything).
		Return(func(input *rolemanager.DestroyRoleWithPolicyInput) *rolemanager.DestroyRoleWithPolicyOutput {
			return &rolemanager.DestroyRoleWithPolicyOutput{
				RoleName:  input.RoleName,
				PolicyArn: input.PolicyArn,
			}
		}, nil)

	return roleManagerMock
}

func createAccountAPIRequest(t *testing.T, req createRequest) *events.APIGatewayProxyRequest {
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

func newCreateController() createController {
	return createController{
		Dao:             dbStub(),
		Queue:           queueStub(),
		SNS:             snsStub(),
		TokenService:    tokenServiceStub(),
		RoleManager:     roleManagerStub(),
		StoragerService: StoragerMock(),
	}
}
