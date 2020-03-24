package accountmanager

import (
	"fmt"
	"log"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type principalService struct {
	iamSvc   iamiface.IAMAPI
	storager common.Storager
	account  *account.Account
	config   ServiceConfig
}

func (p *principalService) MergeRole() error {

	_, err := p.iamSvc.CreateRole(&iam.CreateRoleInput{
		RoleName:                 p.account.PrincipalRoleArn.IAMResourceName(),
		AssumeRolePolicyDocument: aws.String(p.config.assumeRolePolicy),
		Description:              aws.String(p.config.PrincipalRoleDescription),
		MaxSessionDuration:       aws.Int64(p.config.PrincipalMaxSessionDuration),
		Tags: append(p.config.tags,
			&iam.Tag{Key: aws.String("Name"), Value: aws.String("DCEPrincipal")},
		),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + fmt.Sprintf(" (Ignoring this error for account %q)", *p.account.ID))
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error creating role %q", p.account.PrincipalRoleArn.String()), err)
		}
	}

	return nil
}

func (p *principalService) DeleteRole() error {

	_, err := p.iamSvc.DeleteRole(&iam.DeleteRoleInput{
		RoleName: p.account.PrincipalRoleArn.IAMResourceName(),
	})
	if err != nil {
		if isAWSNoSuchEntityError(err) {
			log.Print(err.Error() + fmt.Sprintf(" (Ignoring this error for account %q)", *p.account.ID))
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error deleting the role %q", p.account.PrincipalRoleArn.String()), err)
		}
	}

	return nil
}

func (p *principalService) MergePolicy() error {

	policy, policyHash, err := p.buildPolicy()
	if err != nil {
		return err
	}

	// if they match there is nothing to do
	if p.account.PrincipalPolicyHash != nil {
		if *policyHash == *p.account.PrincipalPolicyHash {
			log.Printf("SKIP: For account %q, Policy Hash matches.  Old %q and New %q", *p.account.ID, *p.account.PrincipalPolicyHash, *policyHash)
			return nil
		}
		log.Printf("UPDATE: For account %q, Policy Hash doesn't match.  Old %q and New %q", *p.account.ID, *p.account.PrincipalPolicyHash, *policyHash)
	} else {
		log.Printf("UPDATE: For account %q, Old Policy Hash is null. New %q", *p.account.ID, *policyHash)
	}

	_, err = p.iamSvc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     p.account.PrincipalPolicyArn.IAMResourceName(),
		Description:    aws.String(p.config.PrincipalPolicyDescription),
		PolicyDocument: policy,
	})

	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + fmt.Sprintf(" (Ignoring this error for account %q)", *p.account.ID))
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error creating policy %q", p.account.PrincipalPolicyArn.String()), err)
		}
	} else {
		// no error means we create the policy without issue moving on
		p.account.PrincipalPolicyHash = policyHash
		return nil
	}

	// Prune old versions of the policy.  Making sure we have room for one more policy version
	err = p.prunePolicyVersions()
	if err != nil {
		return err
	}

	// Create a new Policy Version and set as default
	_, err = p.iamSvc.CreatePolicyVersion(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(p.account.PrincipalPolicyArn.String()),
		PolicyDocument: policy,
		SetAsDefault:   aws.Bool(true),
	})

	p.account.PrincipalPolicyHash = policyHash
	if err != nil {
		return errors.NewInternalServer(fmt.Sprintf("unexpected error creating policy version %q", p.account.PrincipalPolicyArn.String()), err)
	}

	return nil
}

func (p *principalService) DeletePolicy() error {

	versions, err := p.iamSvc.ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
	})
	if err != nil {
		return errors.NewInternalServer(fmt.Sprintf("unexpected error listing policy versions on %q", p.account.PrincipalPolicyArn.String()), err)
	}
	for _, version := range versions.Versions {
		if !*version.IsDefaultVersion {
			err = p.deletePolicyVersion(version)
			if err != nil {
				return err
			}
		}
	}

	_, err = p.iamSvc.DeletePolicy(&iam.DeletePolicyInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
	})

	if err != nil {
		if isAWSNoSuchEntityError(err) {
			log.Print(err.Error() + fmt.Sprintf(" (Ignoring this error for account %q)", *p.account.ID))
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error deleting the policy %q", p.account.PrincipalPolicyArn.String()), err)
		}
	}

	return nil
}

func (p *principalService) AttachRoleWithPolicy() error {

	// Attach the policy to the role
	_, err := p.iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
		RoleName:  p.account.PrincipalRoleArn.IAMResourceName(),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + fmt.Sprintf(" (Ignoring this error for account %q)", *p.account.ID))
		} else {
			return errors.NewInternalServer(
				fmt.Sprintf("unexpected error attaching policy %q to role %q", p.account.PrincipalPolicyArn.String(), p.account.PrincipalRoleArn.String()),
				err)
		}
	}

	return nil
}

func (p *principalService) DetachRoleWithPolicy() error {

	// Detach the policy to the role
	_, err := p.iamSvc.DetachRolePolicy(&iam.DetachRolePolicyInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
		RoleName:  p.account.PrincipalRoleArn.IAMResourceName(),
	})
	if err != nil {
		if isAWSNoSuchEntityError(err) {
			log.Print(err.Error() + fmt.Sprintf(" (Ignoring this error for account %q)", *p.account.ID))
		} else {
			return errors.NewInternalServer(
				fmt.Sprintf("unexpected error detaching policy %q from role %q", p.account.PrincipalPolicyArn.String(), p.account.PrincipalRoleArn.String()),
				err)
		}
	}

	return nil
}

func (p *principalService) buildPolicy() (*string, *string, error) {

	type principalPolicyInput struct {
		PrincipalPolicyArn   string
		PrincipalRoleArn     string
		PrincipalIAMDenyTags []string
		AdminRoleArn         string
		Regions              []string
	}

	policy, policyHash, err := p.storager.GetTemplateObject(p.config.S3BucketName, p.config.S3PolicyKey,
		principalPolicyInput{
			PrincipalPolicyArn:   p.account.PrincipalPolicyArn.String(),
			PrincipalRoleArn:     p.account.PrincipalRoleArn.String(),
			PrincipalIAMDenyTags: p.config.PrincipalIAMDenyTags,
			AdminRoleArn:         p.account.AdminRoleArn.String(),
			Regions:              p.config.AllowedRegions,
		})
	if err != nil {
		return nil, nil, err
	}

	return &policy, &policyHash, nil
}

// PrunePolicyVersions to prune the oldest version if at 5 versions
func (p *principalService) prunePolicyVersions() error {
	versions, err := p.iamSvc.ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
	})
	if err != nil {
		return errors.NewInternalServer(fmt.Sprintf("unexpected error listing policy versions on %q", p.account.PrincipalPolicyArn.String()), err)
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
		return p.deletePolicyVersion(oldestVersion)
	}

	return nil
}

// DeletePolicyVersion delete a version of a template
func (p *principalService) deletePolicyVersion(version *iam.PolicyVersion) error {
	request := &iam.DeletePolicyVersionInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
		VersionId: version.VersionId,
	}

	_, err := p.iamSvc.DeletePolicyVersion(request)
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("unexpected error deleting policy version %q on policy %q", *version.VersionId, p.account.PrincipalPolicyArn.String()),
			err,
		)
	}
	return nil
}
