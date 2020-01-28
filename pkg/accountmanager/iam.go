package accountmanager

import (
	"fmt"
	"log"
	"strings"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/caarlos0/env"
)

type serviceConfig struct {
	AccountID                   string   `env:"ACCOUNT_ID" envDefault:"111111111111"`
	S3BucketName                string   `env:"ARTIFACTS_BUCKET" envDefault:"DefaultArtifactBucket"`
	S3PolicyKey                 string   `env:"PRINCIPAL_POLICY_S3_KEY" envDefault:"DefaultPrincipalPolicyS3Key"`
	PrincipalIAMDenyTags        []string `env:"PRINCIPAL_IAM_DENY_TAGS" envDefault:"DefaultPrincipalIamDenyTags"`
	PrincipalMaxSessionDuration int64    `env:"PRINCIPAL_MAX_SESSION_DURATION" envDefault:"3600"` // 3600 is the default minimum value
	AllowedRegions              []string `env:"ALLOWED_REGIONS" envDefault:"us-east-1"`
	TagEnvironment              string   `env:"TAG_ENVIRONMENT" envDefault:"DefaultTagEnvironment"`
	TagContact                  string   `env:"TAG_CONTACT" envDefault:"DefaultTagContact"`
	TagAppName                  string   `env:"TAG_APP_NAME" envDefault:"DefaultTagAppName"`
	PrincipalRoleDescription    string   `env:"PRINCIPAL_ROLE_DESCRIPTION" envDefault:"Role for principal users of DCE"`
	PrincipalPolicyDescription  string   `env:"PRINCIPAL_POLICY_DESCRIPTION" envDefault:"Policy for principal users of DCE"`
}

var (
	// Config holds static configuration values
	Config serviceConfig
	// Tags has the default IAM Tags
	Tags []*iam.Tag
	// AssumeRolePolicy Default Assume Role Policy
	AssumeRolePolicy string
)

func init() {
	if err := env.Parse(&Config); err != nil {
		panic(err)
	}

	Tags = []*iam.Tag{
		{Key: aws.String("Terraform"), Value: aws.String("False")},
		{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
		{Key: aws.String("Environment"), Value: aws.String(Config.TagEnvironment)},
		{Key: aws.String("Contact"), Value: aws.String(Config.TagContact)},
		{Key: aws.String("AppName"), Value: aws.String(Config.TagAppName)},
	}

	AssumeRolePolicy = strings.TrimSpace(fmt.Sprintf(`
		{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": "arn:aws:iam::%s:root"
					},
					"Action": "sts:AssumeRole",
					"Condition": {}
				}
			]
		}
	`, Config.AccountID))

}

type principalService struct {
	iamSvc   iamiface.IAMAPI
	storager common.Storager
	account  *account.Account
}

func (p *principalService) MergeRole() error {

	_, err := p.iamSvc.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(*p.account.PrincipalRoleName),
		AssumeRolePolicyDocument: aws.String(AssumeRolePolicy),
		Description:              aws.String("Role to be assumed by principal users of DCE"),
		MaxSessionDuration:       aws.Int64(Config.PrincipalMaxSessionDuration),
		Tags: append(Tags,
			&iam.Tag{Key: aws.String("Name"), Value: aws.String("DCEPrincipal")},
		),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error creating role %q", p.account.PrincipalRoleArn.String()), err)
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
	if policyHash == p.account.PrincipalPolicyHash {
		return nil
	}

	_, err = p.iamSvc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     aws.String(iamResourceNameFromArn(*p.account.PrincipalPolicyArn)),
		Description:    aws.String(Config.PrincipalRoleDescription),
		PolicyDocument: policy,
	})

	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return errors.NewInternalServer(fmt.Sprintf("unexpected error creating policy %q", p.account.PrincipalPolicyArn.String()), err)
		}
	} else {
		// no error means we create the policy without issue moving on
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
	if err != nil {
		return errors.NewInternalServer(fmt.Sprintf("unexpected error creating policy version %q", p.account.PrincipalPolicyArn.String()), err)
	}

	return nil
}

func (p *principalService) AttachRoleWithPolicy() error {

	// Attach the policy to the role
	_, err := p.iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: aws.String(p.account.PrincipalPolicyArn.String()),
		RoleName:  aws.String(*p.account.PrincipalRoleName),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return errors.NewInternalServer(
				fmt.Sprintf("unexpected error attaching policy %q to role %q", p.account.PrincipalPolicyArn.String(), p.account.PrincipalRoleArn.String()),
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

	policy, policyHash, err := p.storager.GetTemplateObject(Config.S3BucketName, Config.S3PolicyKey,
		principalPolicyInput{
			PrincipalPolicyArn:   p.account.PrincipalPolicyArn.String(),
			PrincipalRoleArn:     p.account.PrincipalRoleArn.String(),
			PrincipalIAMDenyTags: Config.PrincipalIAMDenyTags,
			AdminRoleArn:         p.account.AdminRoleArn.String(),
			Regions:              Config.AllowedRegions,
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
			fmt.Sprintf("unexpected error deleting policy version on policy %q with version %q", *version.VersionId, p.account.PrincipalPolicyArn.String()),
			err,
		)
	}
	return nil
}
