package accountmanager

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/accountmanager/mocks"
	"github.com/Optum/dce/pkg/arn"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidateAccess(t *testing.T) {

	type assumeRoleOutput struct {
		assumeRoleOutput *sts.AssumeRoleOutput
		err              error
	}

	tests := []struct {
		name       string
		arn        *arn.ARN
		assumeResp assumeRoleOutput
		exp        error
	}{
		{
			name: "should succeed when accessible",
			arn:  arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
			assumeResp: assumeRoleOutput{
				assumeRoleOutput: &sts.AssumeRoleOutput{
					Credentials: &sts.Credentials{
						AccessKeyId:     aws.String("AKID"),
						SecretAccessKey: aws.String("SECRET"),
						SessionToken:    aws.String(""),
						Expiration:      aws.Time(time.Now()),
					},
				},
				err: nil,
			},
		},
		{
			name: "should get an account by ID",
			arn:  arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
			assumeResp: assumeRoleOutput{
				assumeRoleOutput: nil,
				err:              fmt.Errorf("error"),
			},
			exp: errors.NewValidation("account", fmt.Errorf("error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stsSvc := awsMocks.STSAPI{}

			stsSvc.On("AssumeRole", mock.AnythingOfType("*sts.AssumeRoleInput")).Return(tt.assumeResp.assumeRoleOutput, tt.assumeResp.err)
			amSvc, err := NewService(NewServiceInput{
				Sts: &stsSvc,
			})

			assert.Nil(t, err)

			err = amSvc.ValidateAccess(tt.arn)
			assert.True(t, errors.Is(err, tt.exp), "actual error %q doesn't match expected error %q", err, tt.exp)
		})
	}
}

func TestUpsertPrincipalAccess(t *testing.T) {

	type assumeRoleOutput struct {
		output *sts.AssumeRoleOutput
		err    error
	}

	type createRoleOutput struct {
		output *iam.CreateRoleOutput
		err    awserr.Error
	}

	type createPolicyOutput struct {
		output *iam.CreatePolicyOutput
		err    awserr.Error
	}

	type attachRolePolicyOutput struct {
		output *iam.AttachRolePolicyOutput
		err    awserr.Error
	}

	tests := []struct {
		name                   string
		assumeResp             assumeRoleOutput
		exp                    error
		input                  *account.Account
		createRoleOutput       createRoleOutput
		createPolicyOutput     createPolicyOutput
		attachRolePolicyOutput attachRolePolicyOutput
	}{
		{
			name: "should create role and policy and pass",
			input: &account.Account{
				ID:                 aws.String("123456789012"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
				AdminRoleArn:       arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
			},
			assumeResp: assumeRoleOutput{
				output: &sts.AssumeRoleOutput{
					Credentials: &sts.Credentials{
						AccessKeyId:     aws.String("AKID"),
						SecretAccessKey: aws.String("SECRET"),
						SessionToken:    aws.String(""),
						Expiration:      aws.Time(time.Now()),
					},
				},
				err: nil,
			},
			createRoleOutput: createRoleOutput{
				output: &iam.CreateRoleOutput{},
				err:    nil,
			},
			createPolicyOutput: createPolicyOutput{
				output: &iam.CreatePolicyOutput{},
				err:    nil,
			},
			attachRolePolicyOutput: attachRolePolicyOutput{
				output: &iam.AttachRolePolicyOutput{},
				err:    nil,
			},
		},
		{
			name: "when an error is found return the error",
			input: &account.Account{
				ID:                 aws.String("123456789012"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
				AdminRoleArn:       arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
			},
			exp: errors.NewInternalServer("unexpected error creating role \"arn:aws:iam::123456789012:role/DCEPrincipal\"", awserr.New(iam.ErrCodeInvalidInputException, "Conflict", nil)),
			createRoleOutput: createRoleOutput{
				output: nil,
				err:    awserr.New(iam.ErrCodeInvalidInputException, "Conflict", nil),
			},
			createPolicyOutput:     createPolicyOutput{},
			attachRolePolicyOutput: attachRolePolicyOutput{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stsSvc := &awsMocks.STSAPI{}

			stsSvc.On("AssumeRole", mock.AnythingOfType("*sts.AssumeRoleInput")).Return(tt.assumeResp.output, tt.assumeResp.err)

			iamSvc := &awsMocks.IAM{}
			iamSvc.On("CreateRole", mock.Anything).
				Return(tt.createRoleOutput.output, tt.createRoleOutput.err)
			iamSvc.On("CreatePolicy", mock.AnythingOfType("*iam.CreatePolicyInput")).
				Return(tt.createPolicyOutput.output, tt.createPolicyOutput.err)
			iamSvc.On("AttachRolePolicy", mock.AnythingOfType("*iam.AttachRolePolicyInput")).
				Return(tt.attachRolePolicyOutput.output, tt.attachRolePolicyOutput.err)

			storagerSvc := &commonMocks.Storager{}
			storagerSvc.On(
				"GetTemplateObject", "DefaultArtifactBucket", "DefaultPrincipalPolicyS3Key",
				mock.Anything).Return("", "123", nil)

			clientSvc := &mocks.Clienter{}
			clientSvc.On("IAM", mock.Anything).Return(iamSvc)

			amSvc, err := NewService(NewServiceInput{
				Session:  session.Must(session.NewSession()),
				Sts:      stsSvc,
				Storager: storagerSvc,
				Config:   testConfig,
			})
			amSvc.client = clientSvc

			assert.Nil(t, err)

			err = amSvc.UpsertPrincipalAccess(tt.input)
			assert.True(t, errors.Is(err, tt.exp), "actual error %+v doesn't match expected error %+v", err, tt.exp)
		})
	}
}
