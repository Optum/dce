package accountmanager

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/accountmanager/mocks"
	"github.com/Optum/dce/pkg/arn"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

func TestMergePolicyAccess(t *testing.T) {

	type assumeRoleOutput struct {
		assumeRoleOutput *sts.AssumeRoleOutput
		err              error
	}

	tests := []struct {
		name       string
		assumeResp assumeRoleOutput
		exp        error
		input      *account.Account
	}{
		{
			name: "should create role and policy and pass",
			input: &account.Account{
				ID:                aws.String("123456789012"),
				PrincipalRoleArn:  arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
				PrincipalRoleName: aws.String("DCEPrincipal"),
				AdminRoleArn:      arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
			},
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
			input: &account.Account{
				ID:                aws.String("123456789012"),
				AdminRoleArn:      arn.New("aws", "iam", "", "123456789012", "role/AdminAccess"),
				PrincipalRoleArn:  arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
				PrincipalRoleName: aws.String("DCEPrincipal"),
			},
			assumeResp: assumeRoleOutput{
				assumeRoleOutput: nil,
				err:              fmt.Errorf("error"),
			},
			exp: errors.NewValidation("account", fmt.Errorf("error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stsSvc := &awsMocks.STSAPI{}

			stsSvc.On("AssumeRole", mock.AnythingOfType("*sts.AssumeRoleInput")).Return(tt.assumeResp.assumeRoleOutput, tt.assumeResp.err)

			iamSvc := &awsMocks.IAM{}

			clientSvc := &mocks.Clienter{}
			clientSvc.On("IAM", mock.Anything).Return(iamSvc)

			amSvc, err := NewService(NewServiceInput{
				Session: session.Must(session.NewSession()),
				Sts:     stsSvc,
			})
			amSvc.client = clientSvc

			assert.Nil(t, err)

			err = amSvc.MergePrincipalAccess(tt.input)
			assert.True(t, errors.Is(err, tt.exp), "actual error %q doesn't match expected error %q", err, tt.exp)
		})
	}
}
