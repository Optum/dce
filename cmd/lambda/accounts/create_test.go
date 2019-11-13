package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"

	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/rolemanager"
	roleManagerMocks "github.com/Optum/dce/pkg/rolemanager/mocks"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"

	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/db/mocks"
	dbMocks "github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/mock"
)

func TestCreate(t *testing.T) {

	t.Run("should return an account object", func(t *testing.T) {
		// Send request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:*:*:*",
		})

		mockDb := mocks.DBer{}
		mockDb.On("GetAccount", "1234567890").Return(nil, nil)
		mockDb.On("PutAccount", mock.Anything).Return(nil)

		mockAwsSession := &awsMocks.AwsSession{}
		mockAwsSession.On("ClientConfig", mock.Anything).Return(client.Config{
			Config: &aws.Config{},
		})

		mockTokenService := commonMocks.TokenService{}
		mockTokenService.On("AssumeRole", mock.Anything).Return(nil, nil)
		mockTokenService.On("NewSession", mock.Anything, "arn:*:*:*").Return(mockAwsSession, nil)

		mockStorageSvc := commonMocks.Storager{}
		mockStorageSvc.On("GetTemplateObject", mock.Anything, mock.Anything, mock.Anything).Return("Policy", "PolicyHash", nil)

		mockRoleManager := roleManagerMocks.RoleManager{}
		mockRoleManager.On("SetIAMClient", mock.Anything)
		createRoleOutput := &rolemanager.CreateRoleWithPolicyOutput{
			RoleArn:  "arn:*:*:*",
			RoleName: "Role",
		}
		mockRoleManager.On("CreateRoleWithPolicy", mock.Anything).Return(createRoleOutput, nil)

		mockQueue := commonMocks.Queue{}
		mockQueue.On("SendMessage", mock.Anything, mock.Anything).Return(nil)

		mockSns := commonMocks.Notificationer{}
		mockSns.On("PublishMessage", mock.Anything, mock.Anything, true).Return(nil, nil)

		Dao = &mockDb
		TokenSvc = &mockTokenService
		StorageSvc = &mockStorageSvc
		RoleManager = &mockRoleManager
		Queue = &mockQueue
		SnsSvc = &mockSns

		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		// Unmarshal the response JSON into an account object
		resJSON := unmarshal(t, res.Body)

		assert.Equal(t, "1234567890", resJSON["id"])
		assert.Equal(t, "arn:*:*:*", resJSON["adminRoleArn"])
		assert.Equal(t, "NotReady", resJSON["accountStatus"])
		assert.True(t, resJSON["lastModifiedOn"].(float64) > 1561518000)
		assert.True(t, resJSON["createdOn"].(float64) > 1561518000)
	})

	t.Run("should fail if adminRoleArn is missing", func(t *testing.T) {

		mockDb := mocks.DBer{}
		mockDb.On("GetAccount", "1234567890").Return(nil, nil)
		mockDb.On("PutAccount", mock.Anything).Return(nil)

		Dao = &mockDb

		// Send request, missing AdminRoleArn
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		// Check the error response
		assert.Equal(t,
			MockAPIErrorResponse(http.StatusBadRequest, "ClientError", "missing required field \"adminRoleArn\""),
			res,
			"should return a validation error",
		)
	})

	t.Run("should fail if accountID is missing", func(t *testing.T) {
		// Send request, missing AdminRoleArn
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		// Check the error response
		assert.Equal(t,
			MockAPIErrorResponse(http.StatusBadRequest, "ClientError", "missing required field \"id\""),
			res,
			"should return a validation error",
		)
	})

	t.Run("should fail if the adminRoleArn is not assumable", func(t *testing.T) {
		// Configure a controller with a mock token service

		mockDb := mocks.DBer{}
		mockDb.On("GetAccount", "1234567890").Return(nil, nil)

		tokenService := commonMocks.TokenService{}
		// Should fail to assume role
		tokenService.On("AssumeRole",
			mock.MatchedBy(func(input *sts.AssumeRoleInput) bool {
				assert.Equal(t, "arn:iam:adminRole", *input.RoleArn)
				assert.Equal(t, "MasterAssumeRoleVerification", *input.RoleSessionName)

				return true
			}),
		).Return(nil, errors.New("mock error, failed to assume role"))

		defer tokenService.AssertExpectations(t)

		Dao = &mockDb
		TokenSvc = &tokenService

		// Call the controller
		res, err := Handler(
			context.TODO(),
			createAccountAPIRequest(t, CreateRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:iam:adminRole",
			}),
		)
		assert.Nil(t, err)
		assert.Equal(t,
			MockAPIErrorResponse(http.StatusBadRequest, "RequestValidationError", "Unable to add account 1234567890 to pool: adminRole is not assumable by the master account"),
			res,
		)
	})

	t.Run("should add the account to the Account DB Table, as NotReady", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}

		// Mock the DB, so that the account doesn't already exist
		mockDb.On("GetAccount", "1234567890").
			Return(nil, nil)

		// Mock the DB method to create the Account
		mockDb.On("PutAccount",
			mock.MatchedBy(func(account db.Account) bool {
				assert.Equal(t, "1234567890", account.ID)
				assert.Equal(t, "arn:mock", account.AdminRoleArn)
				assert.Equal(t, "arn:aws:iam::1234567890:role/DCEPrincipal", account.PrincipalRoleArn)
				return true
			}),
		).Return(nil)
		defer mockDb.AssertExpectations(t)

		mockAwsSession := &awsMocks.AwsSession{}
		mockAwsSession.On("ClientConfig", mock.Anything).Return(client.Config{
			Config: &aws.Config{},
		})

		mockTokenService := &commonMocks.TokenService{}
		// Should fail to assume role
		mockTokenService.On("AssumeRole",
			mock.MatchedBy(func(input *sts.AssumeRoleInput) bool {
				assert.Equal(t, "arn:mock", *input.RoleArn)
				assert.Equal(t, "MasterAssumeRoleVerification", *input.RoleSessionName)

				return true
			}),
		).Return(nil, nil)
		mockTokenService.On("NewSession", mock.Anything, "arn:mock").Return(mockAwsSession, nil)

		mockRoleManager := &roleManagerMocks.RoleManager{}
		mockRoleManager.On("SetIAMClient", mock.Anything)
		createRoleOutput := &rolemanager.CreateRoleWithPolicyOutput{
			RoleArn:  "arn:aws:iam::1234567890:role/DCEPrincipal",
			RoleName: "Role",
		}
		mockRoleManager.On("CreateRoleWithPolicy", mock.Anything).Return(createRoleOutput, nil)

		Dao = mockDb
		TokenSvc = mockTokenService
		RoleManager = mockRoleManager

		// Send an API request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)
		assert.Equal(t, 201, res.StatusCode)
	})

	t.Run("should return a 409 if the account already exists", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		Dao = mockDb

		// Mock the DB, so that the account already exist
		mockDb.On("GetAccount", "1234567890").
			Return(&db.Account{}, nil)

		// Send an API request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		assert.Equal(t,
			MockAPIErrorResponse(http.StatusConflict, "AlreadyExistsError", "The requested resource cannot be created, as it conflicts with an existing resource"),
			res,
		)
	})

	t.Run("should handle DB.GetAccount response errors as 500s", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		Dao = mockDb

		// Mock the DB to return an error
		mockDb.On("GetAccount", "1234567890").
			Return(nil, errors.New("mock error"))

		// Send an API request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		assert.Equal(t,
			MockAPIErrorResponse(http.StatusInternalServerError, "ServerError", ""),
			res,
		)
	})

	t.Run("should handle DB.PutAccount response errors as 500s", func(t *testing.T) {
		mockDb := &dbMocks.DBer{}
		Dao = mockDb

		// Account doesn't already exist
		mockDb.On("GetAccount", "1234567890").
			Return(nil, nil)

		// Mock the db to return an error
		mockDb.On("PutAccount", mock.Anything).
			Return(errors.New("mock error"))

		// Send an API request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		assert.Equal(t,
			MockAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"),
			res,
		)
	})

	t.Run("should add the account to the reset Queue", func(t *testing.T) {

		Dao = dbStub()
		TokenSvc = tokenServiceStub()
		RoleManager = roleManagerStub()

		// Configure the controller, with a mock SQS
		mockQueue := &commonMocks.Queue{}
		Queue = mockQueue

		queueName := "DefaultResetSQSUrl"
		accountID := "1234567890"

		// Should add account to Queue
		mockQueue.On("SendMessage",
			&queueName,
			&accountID,
		).Return(nil)
		defer mockQueue.AssertExpectations(t)

		// Send request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)
		assert.Equal(t, 201, res.StatusCode, res.Body)
	})

	t.Run("should return a 500, if SQS fails", func(t *testing.T) {
		// Configure the controller, with a mock SQS
		mockQueue := &commonMocks.Queue{}
		mockDB := dbStub()

		// Should fail to add account to Queue
		mockQueue.On("SendMessage",
			mock.Anything,
			mock.Anything,
		).Return(errors.New("mock error"))
		defer mockQueue.AssertExpectations(t)

		Dao = mockDB
		Queue = mockQueue

		// Send request
		req := createAccountAPIRequest(t, CreateRequest{
			ID:           "1234567890",
			AdminRoleArn: "arn:mock",
		})
		res, err := Handler(context.TODO(), req)
		assert.Nil(t, err)

		// Should return an InternalServerError
		assert.Equal(t,
			MockAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"),
			res)

		// Account should still be saved to DB, in `NotReady`
		// state (to be reset later)
		mockDB.AssertCalled(t, "PutAccount", mock.Anything)
	})

	t.Run("should publish an SNS message, with the account info", func(t *testing.T) {

		Dao = dbStub()
		TokenSvc = tokenServiceStub()
		RoleManager = roleManagerStub()
		Queue = queueStub()

		// Configure the controller with mock SNS
		mockSNS := &commonMocks.Notificationer{}
		SnsSvc = mockSNS

		// Expect to publish the account to the SNS topic
		mockSNS.On("PublishMessage",
			mock.MatchedBy(func(arn *string) bool {
				return *arn == "DefaultAccountCreatedTopicArn"
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
				assert.Equal(t, "arn:mock", msgBody["adminRoleArn"])
				assert.Equal(t, "NotReady", msgBody["accountStatus"])
				assert.IsType(t, 0.0, msgBody["lastModifiedOn"])
				assert.IsType(t, 0.0, msgBody["createdOn"])

				return true
			}),
			true,
		).Return(aws.String("mock message"), nil)
		defer mockSNS.AssertExpectations(t)

		// Call the controller with the account
		res, err := Handler(
			context.TODO(),
			createAccountAPIRequest(t, CreateRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mock",
			}),
		)
		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, 201)
	})

	t.Run("should return a 500, if the SNS publish fails", func(t *testing.T) {
		// Configure the controller with mock SNS
		mockSNS := &commonMocks.Notificationer{}
		SnsSvc = mockSNS

		// Mock SNS publish to fail
		mockSNS.On("PublishMessage", mock.Anything, mock.Anything, mock.Anything).
			Return(aws.String(""), errors.New("mock SNS error"))

		// Call the controller with the account
		res, err := Handler(
			context.TODO(),
			createAccountAPIRequest(t, CreateRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mock",
			}),
		)
		assert.Nil(t, err)

		// Should return a ServerError
		assert.Equal(t,
			MockAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"),
			res,
		)
	})

	t.Run("should create a principal role and policy", func(t *testing.T) {

		// Mock the TokenService (assumes role into the user account)
		tokenServiceMock := &commonMocks.TokenService{}
		TokenSvc = tokenServiceMock

		// Mock Token Service, to assume adminRoleArn
		mockAdminRoleSession := &awsMocks.AwsSession{}
		mockAdminRoleSession.On("ClientConfig", mock.Anything).Return(client.Config{
			Config: &aws.Config{},
		})
		tokenServiceMock.On("NewSession", mock.Anything, "arn:mock").
			Return(mockAdminRoleSession, nil)
		tokenServiceMock.On("AssumeRole", mock.Anything).Return(nil, nil)

		// Mock the RoleManager (creates the IAM Role)
		roleManager := roleManagerMocks.RoleManager{}
		RoleManager = &roleManager

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

		// Mock the RoleManager, to create an IAM Role for the Principal user
		roleManager.On("CreateRoleWithPolicy",
			mock.MatchedBy(func(input *rolemanager.CreateRoleWithPolicyInput) bool {
				// Verify the expected input
				assert.Equal(t, "DCEPrincipal", input.RoleName)
				assert.Equal(t, "Role to be assumed by principal users of DCE", input.RoleDescription)
				assert.Equal(t, expectedAssumeRolePolicy, input.AssumeRolePolicyDocument)
				assert.Equal(t, int64(100), input.MaxSessionDuration)
				assert.Equal(t, "DCEPrincipalDefaultPolicy", input.PolicyName)
				assert.Equal(t, []*iam.Tag{
					{Key: aws.String("Terraform"), Value: aws.String("False")},
					{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
					{Key: aws.String("Environment"), Value: aws.String("DefaultTagEnvironment")},
					{Key: aws.String("Contact"), Value: aws.String("DefaultTagContact")},
					{Key: aws.String("AppName"), Value: aws.String("DefaultTagAppName")},
					{Key: aws.String("Name"), Value: aws.String("DCEPrincipal")},
				}, input.Tags)
				assert.Equal(t, true, input.IgnoreAlreadyExistsErrors)
				assert.Equal(t, "", "")

				return true
			}),
		).Return(&rolemanager.CreateRoleWithPolicyOutput{}, nil)

		// Call the controller with the account
		_, err := Handler(
			context.TODO(),
			createAccountAPIRequest(t, CreateRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mock",
			}),
		)
		assert.Nil(t, err)

		roleManager.AssertExpectations(t)
		tokenServiceMock.AssertExpectations(t)
	})

	t.Run("should return a 500 if creating the principal IAM role fails", func(t *testing.T) {
		// Create the controller
		// Mock the RoleManager, to return an error on IAM Role Creation
		roleManager := roleManagerMocks.RoleManager{}
		RoleManager = &roleManager
		roleManager.On("SetIAMClient", mock.Anything)
		roleManager.On("CreateRoleWithPolicy", mock.Anything).
			Return(nil, errors.New("mock error"))

		// Call the controller
		res, err := Handler(
			context.TODO(),
			createAccountAPIRequest(t, CreateRequest{
				ID:           "1234567890",
				AdminRoleArn: "arn:mock",
			}),
		)
		assert.Nil(t, err)

		// Should return a 500 Server Error
		assert.Equal(t,
			MockAPIErrorResponse(http.StatusInternalServerError, "ServerError", "Internal server error"),
			res)
	})

	t.Run("should allow setting metadata", func(t *testing.T) {
		stubAllServices()

		// Mock the DB
		mockDB := &dbMocks.DBer{}
		Dao = mockDB

		// Should write account w/metadata to DB
		mockDB.On("PutAccount",
			mock.MatchedBy(func(acct db.Account) bool {
				assert.Equal(t, acct.Metadata, map[string]interface{}{
					"foo": "bar",
					"faz": "baz",
				})

				return true
			}),
		).Return(nil)

		// stub out other DB methods
		mockDB.On("GetAccount", mock.Anything).
			Return(nil, nil)

		// Call the controller with metadata
		request := createAccountAPIRequest(t, map[string]interface{}{
			"id":           "123456789012",
			"adminRoleArn": "roleArn",
			"metadata": map[string]interface{}{
				"foo": "bar",
				"faz": "baz",
			},
		})
		res, err := Handler(context.TODO(), request)
		require.Nil(t, err)

		// Check the response body
		resJSON := unmarshal(t, res.Body)

		require.Equal(t, map[string]interface{}{
			"foo": "bar",
			"faz": "baz",
		}, resJSON["metadata"])

		mockDB.AssertExpectations(t)
	})

	t.Run("should allow any data type within metadata", func(t *testing.T) {
		stubAllServices()

		// Call the controller with a bunch of different data types
		request := createAccountAPIRequest(t, map[string]interface{}{
			"id":           "123456789012",
			"adminRoleArn": "roleArn",
			"metadata": map[string]interface{}{
				"string": "foobar",
				"int":    7,
				"float":  0.5,
				"bool":   true,
				"obj": map[string]interface{}{
					"nested": map[string]interface{}{
						"object": "value",
					},
				},
				"null": nil,
			},
		})
		res, err := Handler(context.TODO(), request)
		require.Nil(t, err)

		// Check the response body
		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"string": "foobar",
			// something weird with json parsing types here
			// that we have to cast it,
			// but the point is that the API accepted the value, and returned it back
			"int":   float64(7),
			"float": 0.5,
			"bool":  true,
			"obj": map[string]interface{}{
				"nested": map[string]interface{}{
					"object": "value",
				},
			},
			"null": nil,
		}, resJSON["metadata"])
	})

	t.Run("should return an empty metadata JSON object if none is provided", func(t *testing.T) {
		stubAllServices()

		// Call the controller with no metadata param
		request := createAccountAPIRequest(t, map[string]interface{}{
			"id":           "123456789012",
			"adminRoleArn": "roleArn",
		})
		res, err := Handler(context.TODO(), request)
		require.Nil(t, err)

		// Check the response body
		// should return empty JSON object for metadata
		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{}, resJSON["metadata"])
	})

	t.Run("should not allow non-object types for metadata", func(t *testing.T) {
		stubAllServices()

		invalidValues := []interface{}{
			"string",
			14.5,
			true,
		}

		for _, metadata := range invalidValues {
			// Call the controller with no metadata param
			request := createAccountAPIRequest(t, map[string]interface{}{
				"id":           "123456789012",
				"adminRoleArn": "roleArn",
				"metadata":     metadata,
			})
			res, err := Handler(context.TODO(), request)
			require.Nil(t, err)

			// Check the error response
			require.Equalf(t, 400, res.StatusCode, "status code for %v", metadata)
			resJSON := unmarshal(t, res.Body)
			require.Equalf(t, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "ClientError",
					"message": "invalid request parameters",
				},
			}, resJSON, "res JSON for %v", metadata)
		}

	})
}


func createAccountAPIRequest(t *testing.T, req interface{}) events.APIGatewayProxyRequest {
	requestBody, err := json.Marshal(&req)
	assert.Nil(t, err)
	return events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/accounts",
		Body:       string(requestBody),
	}
}