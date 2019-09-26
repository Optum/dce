package rolemanager

import (
	"errors"
	"testing"

	"github.com/Optum/Dce/pkg/awsiface/mocks"
	errors2 "github.com/Optum/Dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateRoleWithPolicy(t *testing.T) {

	t.Run("should create a role, policy, and attachment", func(t *testing.T) {
		// Setup the IAMRoleManager, with mocked IAM client
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock iam.CreateRole()
		mockIAM.On("CreateRole", &iam.CreateRoleInput{
			RoleName:                 aws.String("TestRole"),
			AssumeRolePolicyDocument: aws.String("mock assume role doc"),
			Description:              aws.String("Test Role Description"),
			MaxSessionDuration:       aws.Int64(100),
			Tags: []*iam.Tag{
				{
					Key:   aws.String("Foo"),
					Value: aws.String("Bar"),
				},
			},
		}).Return(&iam.CreateRoleOutput{
			Role: &iam.Role{Arn: aws.String("arn:aws:iam::123456789012:role/MockRole")},
		}, nil)

		// Mock iam.CreatePolicy()
		mockIAM.On("CreatePolicy", &iam.CreatePolicyInput{
			PolicyName:     aws.String("TestPolicy"),
			Description:    aws.String("Test Policy Description"),
			PolicyDocument: aws.String("mock policy document"),
		}).Return(&iam.CreatePolicyOutput{
			Policy: &iam.Policy{
				Arn: aws.String("arn:aws:iam::123456789012:policy/TestPolicy"),
			},
		}, nil)

		// Mock iam.AttachRolePolicy()
		mockIAM.On("AttachRolePolicy", &iam.AttachRolePolicyInput{
			PolicyArn: aws.String("arn:aws:iam::123456789012:policy/TestPolicy"),
			RoleName:  aws.String("TestRole"),
		}).Return(&iam.AttachRolePolicyOutput{}, nil)

		// Call roleManager.CreateRoleWithPolicy()
		res, err := roleManager.CreateRoleWithPolicy(&CreateRoleWithPolicyInput{
			RoleName:                  "TestRole",
			RoleDescription:           "Test Role Description",
			AssumeRolePolicyDocument:  "mock assume role doc",
			MaxSessionDuration:        100,
			PolicyName:                "TestPolicy",
			PolicyDescription:         "Test Policy Description",
			PolicyDocument:            "mock policy document",
			IgnoreAlreadyExistsErrors: false,
			Tags: []*iam.Tag{
				{
					Key:   aws.String("Foo"),
					Value: aws.String("Bar"),
				},
			},
		})
		require.Nil(t, err)
		require.Equal(t, &CreateRoleWithPolicyOutput{
			RoleName:   "TestRole",
			RoleArn:    "arn:aws:iam::123456789012:role/MockRole",
			PolicyName: "TestPolicy",
			PolicyArn:  "arn:aws:iam::123456789012:policy/TestPolicy",
		}, res)
	})

	t.Run("should return role creation errors", func(t *testing.T) {
		// Setup the IAMRoleManager, with mocked IAM client
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock iam.CreateRole() to return an error
		mockIAM.On("CreateRole", mock.Anything).
			Return(nil, errors.New("mock error"))

		// Call roleManager.CreateRoleWithPolicy()
		res, err := roleManager.CreateRoleWithPolicy(&CreateRoleWithPolicyInput{
			RoleName: "TestRole",
		})
		require.Nil(t, res)
		require.NotNil(t, err)
		require.Equal(t, "mock error", err.Error())
	})

	t.Run("should return policy creation errors", func(t *testing.T) {
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock iam.CreateRole()
		mockIAM.On("CreateRole", mock.Anything).
			Return(&iam.CreateRoleOutput{
				Role: &iam.Role{Arn: aws.String("arn:aws:iam::123456789012:role/MockRole")},
			}, nil)

		// Mock iam.CreatePolicy() to return an error
		mockIAM.On("CreatePolicy", mock.Anything).
			Return(nil, errors.New("mock error"))

		// Call roleManager.CreateRoleWithPolicy()
		res, err := roleManager.CreateRoleWithPolicy(&CreateRoleWithPolicyInput{
			RoleName:   "TestRole",
			PolicyName: "TestPolicy",
		})
		require.Nil(t, res)
		require.NotNil(t, err)
		require.Equal(t, "mock error", err.Error())
	})

	t.Run("should return policy attachment errors", func(t *testing.T) {
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock iam.CreateRole()
		mockIAM.On("CreateRole", mock.Anything).
			Return(&iam.CreateRoleOutput{
				Role: &iam.Role{Arn: aws.String("arn:aws:iam::123456789012:role/MockRole")},
			}, nil)

		// Mock iam.CreatePolicy()
		mockIAM.On("CreatePolicy", mock.Anything).
			Return(&iam.CreatePolicyOutput{
				Policy: &iam.Policy{Arn: aws.String("mock:policy:arn")},
			}, nil)

		// Mock iam.AttachRolePolicy() to return an error
		mockIAM.On("AttachRolePolicy", mock.Anything).
			Return(nil, errors.New("mock error"))

		// Call roleManager.CreateRoleWithPolicy()
		res, err := roleManager.CreateRoleWithPolicy(&CreateRoleWithPolicyInput{
			RoleName:   "TestRole",
			PolicyName: "TestPolicy",
		})
		require.Nil(t, res)
		require.NotNil(t, err)
		require.Equal(t, "mock error", err.Error())
	})

	t.Run("should ignore errors AlreadyExists, if IgnoreAlreadyExistsErrors=true", func(t *testing.T) {
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock iam.CreateRole() to return error
		mockIAM.On("CreateRole", mock.Anything).
			Return(nil, AwsAlreadyExistsError{})

		// Mock iam.GetRole() (needed to figure out ARNs of missing resources)
		mockIAM.On("GetRole", mock.Anything).
			Return(&iam.GetRoleOutput{
				Role: &iam.Role{Arn: aws.String("arn:aws:iam::123456789012:role/MockRole")},
			}, nil)

		// Mock iam.CreatePolicy() to return error
		mockIAM.On("CreatePolicy", mock.Anything).
			Return(&iam.CreatePolicyOutput{
				Policy: &iam.Policy{
					Arn: aws.String("arn:aws:iam::123456789012:policy/TestPolicy"),
				},
			},
				nil,
			)

		// Mock iam.AttachRolePolicy() to return an error
		mockIAM.On("AttachRolePolicy", &iam.AttachRolePolicyInput{
			RoleName: aws.String("TestRole"),
			// Note this uses the AWS Account ID from the iam.GetRole response
			PolicyArn: aws.String("arn:aws:iam::123456789012:policy/TestPolicy"),
		}).
			Return(nil, AwsAlreadyExistsError{})

		// Call roleManager.CreateRoleWithPolicy()
		res, err := roleManager.CreateRoleWithPolicy(&CreateRoleWithPolicyInput{
			RoleName:                  "TestRole",
			PolicyName:                "TestPolicy",
			IgnoreAlreadyExistsErrors: true,
		})
		require.Nil(t, err)
		require.NotNil(t, res)
	})

	t.Run("should return other AWS errors, if IgnoreAlreadyExistsErrors=true", func(t *testing.T) {
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock iam.CreateRole() to return a AWS InvalidInput Error
		mockIAM.On("CreateRole", mock.Anything).
			Return(nil, AwsInvalidInputError{})

		// Call roleManager.CreateRoleWithPolicy()
		_, err := roleManager.CreateRoleWithPolicy(&CreateRoleWithPolicyInput{
			RoleName:                  "TestRole",
			PolicyName:                "TestPolicy",
			IgnoreAlreadyExistsErrors: true,
		})
		require.NotNil(t, err)
		require.Equal(t, AwsInvalidInputError{}, err)
	})
}

func TestDestroyRoleWithPolicy(t *testing.T) {
	t.Run("should destroy role with policy", func(t *testing.T) {
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock DetachRolePolicy
		mockIAM.On("DetachRolePolicy", &iam.DetachRolePolicyInput{
			PolicyArn: aws.String("mock:policy:arn"),
			RoleName:  aws.String("MockRoleName"),
		}).Return(&iam.DetachRolePolicyOutput{}, nil)

		// Mock ListPolicyVersions to return two versions (v1, v2)
		mockIAM.On("ListPolicyVersions", &iam.ListPolicyVersionsInput{
			PolicyArn: aws.String("mock:policy:arn"),
		}).Return(&iam.ListPolicyVersionsOutput{
			Versions: []*iam.PolicyVersion{
				{VersionId: aws.String("v1"), IsDefaultVersion: aws.Bool(true)},
				{VersionId: aws.String("v2"), IsDefaultVersion: aws.Bool(false)},
				{VersionId: aws.String("v3"), IsDefaultVersion: aws.Bool(false)},
			},
		}, nil)

		// Mock DeletePolicyVersion for the listed versions (non-defaults
		for _, v := range []string{"v2", "v3"} {
			mockIAM.On("DeletePolicyVersion", &iam.DeletePolicyVersionInput{
				PolicyArn: aws.String("mock:policy:arn"),
				VersionId: aws.String(v),
			}).Return(&iam.DeletePolicyVersionOutput{}, nil)
		}

		// Mock DeletePolicy
		mockIAM.On("DeletePolicy", &iam.DeletePolicyInput{
			PolicyArn: aws.String("mock:policy:arn"),
		}).Return(&iam.DeletePolicyOutput{}, nil)

		// Mock DeleteRole
		mockIAM.On("DeleteRole", &iam.DeleteRoleInput{
			RoleName: aws.String("MockRoleName"),
		}).Return(&iam.DeleteRoleOutput{}, nil)

		// Call roleManager.DestroyRoleWithPolicy()
		res, err := roleManager.DestroyRoleWithPolicy(&DestroyRoleWithPolicyInput{
			RoleName:  "MockRoleName",
			PolicyArn: "mock:policy:arn",
		})
		require.Nil(t, err)
		require.Equal(t, res, &DestroyRoleWithPolicyOutput{
			RoleName:  "MockRoleName",
			PolicyArn: "mock:policy:arn",
		})

		mockIAM.AssertExpectations(t)
	})

	t.Run("should continue on error, and return MultiError", func(t *testing.T) {
		mockIAM := &mocks.IAM{}
		roleManager := IAMRoleManager{
			IAM: mockIAM,
		}

		// Mock DetachRolePolicy (fails)
		mockIAM.On("DetachRolePolicy", &iam.DetachRolePolicyInput{
			PolicyArn: aws.String("mock:policy:arn"),
			RoleName:  aws.String("MockRoleName"),
		}).Return(nil, errors.New("DetachRolePolicy mock failure"))

		// Mock ListPolicyVersions to return two versions (v1, v2)
		mockIAM.On("ListPolicyVersions", &iam.ListPolicyVersionsInput{
			PolicyArn: aws.String("mock:policy:arn"),
		}).Return(&iam.ListPolicyVersionsOutput{
			Versions: []*iam.PolicyVersion{
				{VersionId: aws.String("v1"), IsDefaultVersion: aws.Bool(true)},
				{VersionId: aws.String("v2"), IsDefaultVersion: aws.Bool(false)},
				{VersionId: aws.String("v3"), IsDefaultVersion: aws.Bool(false)},
			},
		}, nil)

		// Mock DeletePolicyVersion for the listed versions
		for _, v := range []string{"v2", "v3"} {
			mockIAM.On("DeletePolicyVersion", &iam.DeletePolicyVersionInput{
				PolicyArn: aws.String("mock:policy:arn"),
				VersionId: aws.String(v),
			}).Return(&iam.DeletePolicyVersionOutput{}, nil)
		}

		// Mock DeletePolicy (fails)
		mockIAM.On("DeletePolicy", &iam.DeletePolicyInput{
			PolicyArn: aws.String("mock:policy:arn"),
		}).Return(nil, errors.New("DeletePolicy mock failure"))

		// Mock DeleteRole
		mockIAM.On("DeleteRole", &iam.DeleteRoleInput{
			RoleName: aws.String("MockRoleName"),
		}).Return(&iam.DeleteRoleOutput{}, nil)

		// Call roleManager.DestroyRoleWithPolicy()
		res, err := roleManager.DestroyRoleWithPolicy(&DestroyRoleWithPolicyInput{
			RoleName:  "MockRoleName",
			PolicyArn: "mock:policy:arn",
		})
		require.Nil(t, res)
		require.Equal(t, err, errors2.NewMultiError(
			"Failed to destroy role MockRoleName and policy mock:policy:arn",
			[]error{errors.New("DetachRolePolicy mock failure"), errors.New("DeletePolicy mock failure")},
		))

		mockIAM.AssertExpectations(t)
	})
}

type AwsAlreadyExistsError struct {
}

func (AwsAlreadyExistsError) Error() string {
	return "mock AWSAlreadyExistsError"
}

func (AwsAlreadyExistsError) Code() string {
	return iam.ErrCodeEntityAlreadyExistsException
}

func (AwsAlreadyExistsError) Message() string {
	return "mock AWSAlreadyExistsError"
}

func (AwsAlreadyExistsError) OrigErr() error {
	return errors.New("mock original error")
}

type AwsInvalidInputError struct {
}

func (AwsInvalidInputError) Error() string {
	return "mock AwsInvalidInputError"
}

func (AwsInvalidInputError) Code() string {
	return iam.ErrCodeInvalidInputException
}

func (AwsInvalidInputError) Message() string {
	return "mock AwsInvalidInputError"
}

func (AwsInvalidInputError) OrigErr() error {
	return errors.New("mock original error")
}
