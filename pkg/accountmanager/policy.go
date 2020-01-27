package accountmanager

import (
	"fmt"
	"log"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type mergePolicyInput struct {
	iam         iamiface.IAMAPI
	policyArn   arn.ARN
	description *string
	document    *string
}

// mergePolicy creates or updates the policy
func mergePolicy(input *mergePolicyInput) error {

	policy, err := input.iam.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     aws.String(iamResourceNameFromArn(input.policyArn)),
		Description:    input.description,
		PolicyDocument: input.document,
	})

	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error creating policy %q", input.policyArn), err)
		}
	} else {
		// no error means we create the policy without issue moving on
		return nil
	}

	// Prune old versions of the policy.  Making sure we have room for one more policy version
	err = prunePolicyVersions(input.iam, input.policyArn)
	if err != nil {
		return err
	}

	// Create a new Policy Version and set as default
	_, err = input.iam.CreatePolicyVersion(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(input.policyArn.String()),
		PolicyDocument: input.document,
		SetAsDefault:   aws.Bool(true),
	})
	if err != nil {
		log.Printf("Found an issue creating a new policy version for policy '%s': %s", *policy.Policy.Arn, err)
		return err
	}

	return nil
}

// PrunePolicyVersions to prune the oldest version if at 5 versions
func prunePolicyVersions(iamAPI iamiface.IAMAPI, policyArn arn.ARN) error {
	versions, err := iamAPI.ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(policyArn.String()),
	})
	if err != nil {
		return errors.NewInternalServer(fmt.Sprintf("unexpected error listing policy versions on %q", policyArn.String()), err)
	}
	if len(versions.Versions) < 5 && len(versions.Versions) > 1 {
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

	if oldestVersion != nil {
		return deletePolicyVersion(iamAPI, policyArn, oldestVersion)
	}

	return nil
}

// DeletePolicyVersion delete a version of a template
func deletePolicyVersion(iamAPI iamiface.IAMAPI, policyArn arn.ARN, version *iam.PolicyVersion) error {
	request := &iam.DeletePolicyVersionInput{
		PolicyArn: aws.String(policyArn.String()),
		VersionId: version.VersionId,
	}

	_, err := iamAPI.DeletePolicyVersion(request)
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("unexpected error deleting policy version on policy %q with version %q", *version.VersionId, policyArn.String()),
			err,
		)
	}
	return nil
}
