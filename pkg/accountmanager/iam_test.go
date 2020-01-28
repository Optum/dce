package accountmanager

import (
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/accountmanager/mocks"
	"github.com/Optum/dce/pkg/arn"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrincipalMergePolicyAccess(t *testing.T) {

	type createPolicyOutput struct {
		output *iam.CreatePolicyOutput
		err    awserr.Error
	}

	type attachRolePolicyOutput struct {
		output *iam.AttachRolePolicyOutput
		err    awserr.Error
	}

	type listPolicyVersionsOutput struct {
		output *iam.ListPolicyVersionsOutput
		err    error
	}
	type createPolicyVersionOutput struct {
		output *iam.CreatePolicyVersionOutput
		err    error
	}

	tests := []struct {
		name                      string
		exp                       error
		account                   *account.Account
		createPolicyOutput        createPolicyOutput
		attachRolePolicyOutput    attachRolePolicyOutput
		listPolicyVersionsOutput  listPolicyVersionsOutput
		createPolicyVersionOutput createPolicyVersionOutput
	}{
		{
			name: "should create role and policy and pass",
			account: &account.Account{
				ID:                 aws.String("123456789012"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
				PrincipalRoleName:  aws.String("DCEPrincipal"),
				AdminRoleArn:       arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
			},
			createPolicyOutput: createPolicyOutput{
				output: &iam.CreatePolicyOutput{},
				err:    nil,
			},
			attachRolePolicyOutput: attachRolePolicyOutput{
				output: &iam.AttachRolePolicyOutput{},
				err:    nil,
			},
			listPolicyVersionsOutput: listPolicyVersionsOutput{},
		},
		{
			name: "should get duplicate errors and still work",
			account: &account.Account{
				ID:                 aws.String("123456789012"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
				PrincipalRoleName:  aws.String("DCEPrincipal"),
				AdminRoleArn:       arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
			},
			exp: nil,
			createPolicyOutput: createPolicyOutput{
				output: nil,
				err:    awserr.New(iam.ErrCodeEntityAlreadyExistsException, "Already Exists", nil),
			},
			listPolicyVersionsOutput: listPolicyVersionsOutput{
				output: &iam.ListPolicyVersionsOutput{},
				err:    nil,
			},
			createPolicyVersionOutput: createPolicyVersionOutput{
				output: &iam.CreatePolicyVersionOutput{},
				err:    nil,
			},
			attachRolePolicyOutput: attachRolePolicyOutput{
				output: nil,
				err:    awserr.New(iam.ErrCodeEntityAlreadyExistsException, "Already Exists", nil),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iamSvc := &awsMocks.IAM{}
			iamSvc.On("CreatePolicy", mock.AnythingOfType("*iam.CreatePolicyInput")).
				Return(tt.createPolicyOutput.output, tt.createPolicyOutput.err)
			iamSvc.On("ListPolicyVersions", mock.AnythingOfType("*iam.ListPolicyVersionsInput")).
				Return(tt.listPolicyVersionsOutput.output, tt.listPolicyVersionsOutput.err)
			iamSvc.On("CreatePolicyVersion", mock.AnythingOfType("*iam.CreatePolicyVersionInput")).
				Return(tt.createPolicyVersionOutput.output, tt.createPolicyVersionOutput.err)
			iamSvc.On("AttachRolePolicy", mock.AnythingOfType("*iam.AttachRolePolicyInput")).
				Return(tt.attachRolePolicyOutput.output, tt.attachRolePolicyOutput.err)

			storagerSvc := &commonMocks.Storager{}
			storagerSvc.On(
				"GetTemplateObject", "DefaultArtifactBucket", "DefaultPrincipalPolicyS3Key",
				mock.Anything).Return("", "123", nil)

			clientSvc := &mocks.Clienter{}
			clientSvc.On("IAM", mock.Anything).Return(iamSvc)

			principalSvc := principalService{
				iamSvc:   iamSvc,
				storager: storagerSvc,
				account:  tt.account,
			}

			err := principalSvc.MergePolicy()
			assert.True(t, errors.Is(err, tt.exp), "actual error %+v doesn't match expected error %+v", err, tt.exp)
		})
	}
}
