package rolemanager

import (
	"fmt"
	"log"

	"github.com/Optum/Dce/pkg/awsiface"
	errors2 "github.com/Optum/Dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/aws/aws-sdk-go/aws"
)

type RoleManager interface {
	CreateRoleWithPolicy(input *CreateRoleWithPolicyInput) (*CreateRoleWithPolicyOutput, error)
	SetIAMClient(iamClient awsiface.IAM)
	DestroyRoleWithPolicy(input *DestroyRoleWithPolicyInput) (*DestroyRoleWithPolicyOutput, *errors2.MultiError)
}

type Policy struct {
	Name               string
	RolePolicyDocument string
	RolePolicyArn      string
	Description        string
}

type IAMRoleManager struct {
	IAM awsiface.IAM
}

type CreateRoleWithPolicyInput struct {
	RoleName                 string
	RoleDescription          string
	AssumeRolePolicyDocument string
	MaxSessionDuration       int64
	PolicyName               string
	PolicyDocument           string
	PolicyDescription        string
	Tags                     []*iam.Tag
	// If false, method will fail if the role/policy/attachment already exists.
	// If true, these errors will be logged and ignored
	IgnoreAlreadyExistsErrors bool
}

type CreateRoleWithPolicyOutput struct {
	RoleName   string
	RoleArn    string
	PolicyName string
	PolicyArn  string
}

// Create a Role, and attach a policy to it
func (rm *IAMRoleManager) CreateRoleWithPolicy(input *CreateRoleWithPolicyInput) (*CreateRoleWithPolicyOutput, error) {
	// Create the IAM Role
	createRoleRes, err := rm.IAM.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(input.RoleName),
		AssumeRolePolicyDocument: aws.String(input.AssumeRolePolicyDocument),
		Description:              aws.String(input.RoleDescription),
		MaxSessionDuration:       aws.Int64(input.MaxSessionDuration),
		Tags:                     input.Tags,
	})
	var roleArn *string
	if err != nil {
		if isAWSAlreadyExistsError(err) && input.IgnoreAlreadyExistsErrors {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return nil, err
		}

		// Lookup the RoleArn
		getRoleRes, err := rm.IAM.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(input.RoleName),
		})
		if err != nil {
			return nil, err
		}
		roleArn = getRoleRes.Role.Arn
	} else {
		roleArn = createRoleRes.Role.Arn
	}

	// Lookup the Account ID, from the RoleArn
	roleArnObj, err := arn.Parse(*roleArn)
	if err != nil {
		return nil, err
	}
	accountID := roleArnObj.AccountID

	// Create the policy
	pm := IAMPolicyManager{}
	pm.SetIAMClient(rm.IAM)
	policyArn := arn.ARN{}
	policyArn, err = arn.Parse(fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, input.PolicyName))
	if err != nil {
		return nil, err
	}
	policyArnString := policyArn.String()
	err = pm.MergePolicy(&MergePolicyInput{
		PolicyArn:         policyArn,
		PolicyName:        input.PolicyName,
		PolicyDocument:    input.PolicyDocument,
		PolicyDescription: input.PolicyDescription,
	})
	if err != nil {
		return nil, err
	}

	// Attach the policy to the role
	_, err = rm.IAM.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: &policyArnString,
		RoleName:  aws.String(input.RoleName),
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) && input.IgnoreAlreadyExistsErrors {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return nil, err
		}
	}

	return &CreateRoleWithPolicyOutput{
		RoleName:   input.RoleName,
		RoleArn:    *roleArn,
		PolicyName: input.PolicyName,
		PolicyArn:  policyArnString,
	}, nil
}

// SetSession allows consumer to set IAM session in IAMRoleManager stuct
func (rm *IAMRoleManager) SetIAMClient(iamClient awsiface.IAM) {
	rm.IAM = iamClient
}

type DestroyRoleWithPolicyInput struct {
	RoleName  string
	PolicyArn string
}

type DestroyRoleWithPolicyOutput struct {
	RoleName  string
	PolicyArn string
}

func (rm *IAMRoleManager) DestroyRoleWithPolicy(input *DestroyRoleWithPolicyInput) (*DestroyRoleWithPolicyOutput, *errors2.MultiError) {
	errs := []error{}
	// Detach the Policy from the Role
	_, err := rm.IAM.DetachRolePolicy(&iam.DetachRolePolicyInput{
		PolicyArn: aws.String(input.PolicyArn),
		RoleName:  aws.String(input.RoleName),
	})
	// If error, continue and return later
	if err != nil {
		errs = append(errs, err)
	}

	// Delete the policy
	err = rm.deletePolicyWithVersion(input.PolicyArn)
	if err != nil {
		errs = append(errs, err)
	}

	// Delete the role
	_, err = rm.IAM.DeleteRole(&iam.DeleteRoleInput{
		RoleName: aws.String(input.RoleName),
	})
	if err != nil {
		errs = append(errs, err)
	}

	// Handle any errors
	if len(errs) != 0 {
		return nil, errors2.NewMultiError(
			fmt.Sprintf("Failed to destroy role %s and policy %s", input.RoleName, input.PolicyArn),
			errs,
		)
	}

	return &DestroyRoleWithPolicyOutput{
		RoleName:  input.RoleName,
		PolicyArn: input.PolicyArn,
	}, nil
}

func (rm *IAMRoleManager) deletePolicyWithVersion(policyArn string) error {
	// List all policy versions
	policyListRes, err := rm.IAM.ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return err
	}

	// Delete all policy versions
	for _, version := range policyListRes.Versions {
		// You cannot delete default policy version
		if !*version.IsDefaultVersion {
			_, err := rm.IAM.DeletePolicyVersion(&iam.DeletePolicyVersionInput{
				PolicyArn: aws.String(policyArn),
				VersionId: version.VersionId,
			})
			if err != nil {
				return err
			}
		}
	}

	// Delete the policy
	_, err = rm.IAM.DeletePolicy(&iam.DeletePolicyInput{
		PolicyArn: aws.String(policyArn),
	})
	return err
}
