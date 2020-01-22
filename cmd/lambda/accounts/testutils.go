package main

import (
	"encoding/json"
	"fmt"
	"testing"

	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/rolemanager"
	roleManagerMocks "github.com/Optum/dce/pkg/rolemanager/mocks"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/stretchr/testify/assert"

	"github.com/Optum/dce/pkg/common"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	dbMocks "github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/mock"
)

func unmarshal(t *testing.T, jsonStr string) map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.Nil(t, err,
		fmt.Sprintf("Failed to unmarshal JSON: %s; %s", jsonStr, err),
	)

	return data
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
		Return(func(accountID string) *db.Account {
			return &db.Account{ID: accountID}
		}, nil)
	mockDb.On("UpdateAccount", mock.Anything, mock.Anything).
		Return(func(acct db.Account, fields []string) *db.Account {
			return &acct
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

func storageStub() common.Storager {
	storagerMock := &commonMocks.Storager{}

	storagerMock.On("GetTemplateObject", mock.Anything, mock.Anything, mock.Anything).
		Return("", "", nil)

	return storagerMock
}

func stubAllServices() {
	TokenSvc = tokenServiceStub()
	RoleManager = roleManagerStub()
	StorageSvc = storageStub()
	Queue = queueStub()
	Dao = dbStub()
	SnsSvc = snsStub()
}

func roleManagerStub() *roleManagerMocks.RoleManager {
	roleManagerMock := &roleManagerMocks.RoleManager{}
	roleManagerMock.On("SetIAMClient", mock.Anything)
	roleManagerMock.On("CreateRoleWithPolicy", mock.Anything).
		Return(
			func(input *rolemanager.CreateRoleWithPolicyInput) *rolemanager.CreateRoleWithPolicyOutput {
				return &rolemanager.CreateRoleWithPolicyOutput{
					RoleName:   input.RoleName,
					RoleArn:    "arn:aws:iam::1234567890:role/" + input.RoleName,
					PolicyName: "DCEPrincipalDefaultPolicy",
					PolicyArn:  "arn:aws:iam::1234567890:policy/DCEPrincipalDefaultPolicy",
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
