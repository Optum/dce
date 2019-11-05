package main

import (
	"fmt"
	"testing"

	"github.com/Optum/Redbox/pkg/rolemanager"

	awsMocks "github.com/Optum/Redbox/pkg/awsiface/mocks"
	commonmock "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbmock "github.com/Optum/Redbox/pkg/db/mocks"
	roleMock "github.com/Optum/Redbox/pkg/rolemanager/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testTransitionFinanceLockInput is the structured input for testing the helper
// function transitionFinanceLock
type testUpdateRedboxPrincipalPolicy struct {
	ExpectedError              error
	GetAccountResult           *db.Account
	GetAccountError            error
	TransitionLeaseStatusError error
	PrincipalPolicyName        string
	PrincipalRoleName          string
	PrincipalPolicyHash        string
	PrincipalIAMDenyTags       []string
	StoragerPolicy             string
	StoragerError              error
}

func TestUpdateRedboxPrincipalPolicy(t *testing.T) {

	tests := []testUpdateRedboxPrincipalPolicy{
		// Happy Path Update Principal Policy
		{
			GetAccountResult: &db.Account{
				ID:           "123456789012",
				AdminRoleArn: "arn:aws:iam::123456789012:role/RedBoxAdminRole",
			},
			PrincipalPolicyName:  "RedboxPrincipalPolicy",
			PrincipalRoleName:    "RedboxPrincipalRole",
			PrincipalPolicyHash:  "aHash",
			PrincipalIAMDenyTags: []string{"Redbox"},
			StoragerPolicy:       "{\"Test\" : \"Policy\"}",
		},
		// Same hash exists don't update.
		{
			GetAccountResult: &db.Account{
				ID:                  "123456789012",
				AdminRoleArn:        "arn:aws:iam::123456789012:role/RedBoxAdminRole",
				PrincipalPolicyHash: "aHash",
			},
			PrincipalPolicyName:  "RedboxPrincipalPolicy",
			PrincipalRoleName:    "RedboxPrincipalRole",
			PrincipalPolicyHash:  "aHash",
			PrincipalIAMDenyTags: []string{"Redbox"},
			StoragerPolicy:       "{\"Test\" : \"Policy\"}",
		},
	}

	// Iterate through each test in the list
	for _, test := range tests {
		// Setup mocks
		mockDB := dbmock.DBer{}
		mockDB.On("GetAccount", mock.Anything).Return(
			test.GetAccountResult,
			test.GetAccountError)
		mockDB.On("UpdateAccountPrincipalPolicyHash",
			test.GetAccountResult.ID,
			test.GetAccountResult.PrincipalPolicyHash,
			test.PrincipalPolicyHash,
		).Return(nil, nil)
		mockS3 := &commonmock.Storager{}
		mockS3.On("GetTemplateObject", mock.Anything, mock.Anything, getPolicyInput{
			PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", test.GetAccountResult.ID, test.PrincipalPolicyName),
			PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", test.GetAccountResult.ID, test.PrincipalRoleName),
			PrincipalIAMDenyTags: test.PrincipalIAMDenyTags,
			AdminRoleArn:         test.GetAccountResult.AdminRoleArn,
		}).Return(
			test.StoragerPolicy,
			test.PrincipalPolicyHash,
			test.StoragerError,
		)

		mockAdminRoleSession := &awsMocks.AwsSession{}
		mockToken := &commonmock.TokenService{}
		mockRoleManager := &roleMock.PolicyManager{}
		mockSession := &awsMocks.AwsSession{}
		if test.PrincipalPolicyHash != test.GetAccountResult.PrincipalPolicyHash {
			mockAdminRoleSession.On("ClientConfig", mock.Anything).Return(client.Config{
				Config: &aws.Config{},
			})
			mockToken.On("NewSession", mock.Anything, test.GetAccountResult.AdminRoleArn).
				Return(mockAdminRoleSession, nil)
			mockToken.On("AssumeRole", mock.Anything).Return(nil, nil)

			mockRoleManager.On("SetIAMClient", mock.Anything).Return()
			policyArn, _ := arn.Parse(fmt.Sprintf("arn:aws:iam::%s:policy/%s", test.GetAccountResult.ID, test.PrincipalPolicyName))
			mockRoleManager.On("MergePolicy", &rolemanager.MergePolicyInput{
				PolicyArn:      policyArn,
				PolicyName:     test.PrincipalPolicyName,
				PolicyDocument: test.StoragerPolicy,
			}).Return(nil)

		}

		// Call transitionFinanceLock
		err := processRecord(processRecordInput{
			AccountID:            test.GetAccountResult.ID,
			DbSvc:                &mockDB,
			StoragerSvc:          mockS3,
			TokenSvc:             mockToken,
			AwsSession:           mockSession,
			RoleManager:          mockRoleManager,
			PrincipalRoleName:    test.PrincipalRoleName,
			PrincipalPolicyName:  test.PrincipalPolicyName,
			PrincipalIAMDenyTags: test.PrincipalIAMDenyTags,
		})

		// Assert expectations
		if test.ExpectedError != nil {
			require.Equal(t, test.ExpectedError.Error(), err.Error())
		} else {
			require.Nil(t, err)
		}
	}
}
