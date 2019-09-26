package rolemanager

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/Optum/Redbox/pkg/awsiface"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

// PolicyManager updates and manages policy versions
type PolicyManager interface {
	MergePolicy(input *MergePolicyInput) error
	SetIAMClient(iamClient awsiface.IAM)
	PrunePolicyVersions(arn string) error
	DeletePolicyVersion(arn string, versionID string) error
}

// MergePolicyInput defines what we need to create and upate a policy
type MergePolicyInput struct {
	PolicyArn         arn.ARN
	PolicyName        string
	PolicyDocument    string
	PolicyDescription string
}

// IAMPolicyManager has the interface to the AWS Session
type IAMPolicyManager struct {
	IAM awsiface.IAM
}

// SetIAMClient allows consumer to set IAM session in IAMRoleManager stuct
func (rm *IAMPolicyManager) SetIAMClient(iamClient awsiface.IAM) {
	rm.IAM = iamClient
}

// MergePolicy creates or updates the policy
func (rm *IAMPolicyManager) MergePolicy(input *MergePolicyInput) error {
	// Check if Policy Already exists
	_, err := rm.IAM.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     aws.String(input.PolicyName),
		Description:    aws.String(input.PolicyDescription),
		PolicyDocument: aws.String(input.PolicyDocument),
	})

	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return err
		}
	} else {
		// No error we create the policy with the correction version
		return nil
	}

	// Prune old versions of the policy.  Making sure we have room for one more policy version
	err = rm.PrunePolicyVersions(input.PolicyArn.String())
	if err != nil {
		log.Printf("Found an issue pruning versions for policy '%s': %s", input.PolicyArn.String(), err)
		return err
	}

	// Create a new Policy Version and set as default
	_, err = rm.IAM.CreatePolicyVersion(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(input.PolicyArn.String()),
		PolicyDocument: aws.String(input.PolicyDocument),
		SetAsDefault:   aws.Bool(true),
	})
	if err != nil {
		log.Printf("Found an issue creating a new policy version for policy '%s': %s", input.PolicyArn.String(), err)
		return err
	}

	return nil
}

// PrunePolicyVersions to prune the oldest version if at 5 versions
func (rm *IAMPolicyManager) PrunePolicyVersions(arn string) error {
	versions, err := rm.IAM.ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(arn),
	})
	if err != nil {
		return err
	}
	if len(versions.Versions) < 5 {
		return nil
	}

	var oldestVersion *iam.PolicyVersion

	for _, version := range versions.Versions {
		if *version.IsDefaultVersion {
			continue
		}
		if oldestVersion == nil ||
			version.CreateDate.Before(*oldestVersion.CreateDate) {
			oldestVersion = version
		}
	}

	err1 := rm.DeletePolicyVersion(arn, *oldestVersion.VersionId)
	return err1
}

// DeletePolicyVersion delete a version of a template
func (rm *IAMPolicyManager) DeletePolicyVersion(arn string, versionID string) error {
	request := &iam.DeletePolicyVersionInput{
		PolicyArn: aws.String(arn),
		VersionId: aws.String(versionID),
	}

	_, err := rm.IAM.DeletePolicyVersion(request)
	if err != nil {
		return fmt.Errorf("Error deleting version %s from IAM policy %s: %s", versionID, arn, err)
	}
	return nil
}
