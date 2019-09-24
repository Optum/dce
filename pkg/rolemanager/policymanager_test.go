package rolemanager

import (
	"testing"
	"time"

	"github.com/Optum/Dcs/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestCreatePolicyManager(t *testing.T) {

	t.Run("should create a policy", func(t *testing.T) {
		// Setup the IAMRoleManager, with mocked IAM client
		mockIAM := &mocks.IAM{}
		roleManager := IAMPolicyManager{
			IAM: mockIAM,
		}

		// Mock iam.CreatePolicy()
		mockIAM.On("CreatePolicy", &iam.CreatePolicyInput{
			PolicyName:     aws.String("name"),
			Description:    aws.String("description"),
			PolicyDocument: aws.String("document"),
		}).Return(nil, nil)

		// Call roleManager.CreateRoleWithPolicy()
		policyArn, _ := arn.Parse("arn:aws:iam::123456789012:policy/name")
		err := roleManager.MergePolicy(&MergePolicyInput{
			PolicyArn:         policyArn,
			PolicyName:        "name",
			PolicyDocument:    "document",
			PolicyDescription: "description",
		})
		require.Nil(t, err)
	})

	t.Run("should continue on duplicate", func(t *testing.T) {
		// Setup the IAMRoleManager, with mocked IAM client
		mockIAM := &mocks.IAM{}
		roleManager := IAMPolicyManager{
			IAM: mockIAM,
		}
		policyArnString := "arn:aws:iam::123456789012:policy/name"
		policyArn, _ := arn.Parse(policyArnString)

		newPolicyDocument := "document"
		// Mock iam.CreatePolicy()
		mockIAM.On("CreatePolicy", &iam.CreatePolicyInput{
			PolicyName:     aws.String("name"),
			Description:    aws.String("description"),
			PolicyDocument: aws.String(newPolicyDocument),
		}).Return(nil, AwsAlreadyExistsError{})

		// mock list policy versions
		policyDocument := "existing document"
		policyVersion := &iam.PolicyVersion{
			Document: &policyDocument,
		}
		mockIAM.On("ListPolicyVersions", &iam.ListPolicyVersionsInput{
			PolicyArn: &policyArnString,
		}).Return(
			&iam.ListPolicyVersionsOutput{
				Versions: []*iam.PolicyVersion{policyVersion},
			},
			nil,
		)

		// mock Create Policy Version
		setAsDefault := true
		mockIAM.On("CreatePolicyVersion", &iam.CreatePolicyVersionInput{
			PolicyArn:      &policyArnString,
			PolicyDocument: &newPolicyDocument,
			SetAsDefault:   &setAsDefault,
		}).Return(
			&iam.CreatePolicyVersionOutput{
				PolicyVersion: &iam.PolicyVersion{},
			}, nil,
		)

		err := roleManager.MergePolicy(&MergePolicyInput{
			PolicyArn:         policyArn,
			PolicyName:        "name",
			PolicyDocument:    "document",
			PolicyDescription: "description",
		})
		require.Nil(t, err)
	})

	t.Run("should continue and prun on 5 policies", func(t *testing.T) {
		// Setup the IAMRoleManager, with mocked IAM client
		mockIAM := &mocks.IAM{}
		roleManager := IAMPolicyManager{
			IAM: mockIAM,
		}
		policyArnString := "arn:aws:iam::123456789012:policy/name"
		policyArn, _ := arn.Parse(policyArnString)

		newPolicyDocument := "document"
		// Mock iam.CreatePolicy()
		mockIAM.On("CreatePolicy", &iam.CreatePolicyInput{
			PolicyName:     aws.String("name"),
			Description:    aws.String("description"),
			PolicyDocument: aws.String(newPolicyDocument),
		}).Return(nil, AwsAlreadyExistsError{})

		// mock list policy versions
		policyDocument := "existing document"
		firstDate := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		lastDate := time.Date(2009, 11, 16, 20, 34, 58, 651387237, time.UTC)
		newIsDefault := true
		oldIsDefault := false
		versionID1 := "1"
		versionID2 := "2"
		versionID3 := "3"
		versionID4 := "4"
		versionID5 := "5"
		mockIAM.On("ListPolicyVersions", &iam.ListPolicyVersionsInput{
			PolicyArn: &policyArnString,
		}).Return(
			&iam.ListPolicyVersionsOutput{
				Versions: []*iam.PolicyVersion{
					&iam.PolicyVersion{
						Document:         &policyDocument,
						CreateDate:       &firstDate,
						IsDefaultVersion: &newIsDefault,
						VersionId:        &versionID1,
					},
					&iam.PolicyVersion{
						Document:         &policyDocument,
						CreateDate:       &firstDate,
						IsDefaultVersion: &oldIsDefault,
						VersionId:        &versionID2,
					},
					&iam.PolicyVersion{
						Document:         &policyDocument,
						CreateDate:       &firstDate,
						IsDefaultVersion: &oldIsDefault,
						VersionId:        &versionID3,
					},
					&iam.PolicyVersion{
						Document:         &policyDocument,
						CreateDate:       &firstDate,
						IsDefaultVersion: &oldIsDefault,
						VersionId:        &versionID4,
					},
					&iam.PolicyVersion{
						Document:         &policyDocument,
						CreateDate:       &lastDate,
						IsDefaultVersion: &oldIsDefault,
						VersionId:        &versionID5,
					},
				},
			},
			nil,
		)
		mockIAM.On("DeletePolicyVersion", &iam.DeletePolicyVersionInput{
			PolicyArn: &policyArnString,
			VersionId: &versionID5,
		}).Return(nil, nil)

		// mock Create Policy Version
		setAsDefault := true
		mockIAM.On("CreatePolicyVersion", &iam.CreatePolicyVersionInput{
			PolicyArn:      &policyArnString,
			PolicyDocument: &newPolicyDocument,
			SetAsDefault:   &setAsDefault,
		}).Return(
			nil, nil,
		)

		err := roleManager.MergePolicy(&MergePolicyInput{
			PolicyArn:         policyArn,
			PolicyName:        "name",
			PolicyDocument:    "document",
			PolicyDescription: "description",
		})
		require.Nil(t, err)
	})
}
