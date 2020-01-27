package main

import (
	"github.com/Optum/dce/pkg/rolemanager"
	roleManagerMocks "github.com/Optum/dce/pkg/rolemanager/mocks"

	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/mock"
)

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
