package accountmanager

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
)

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

// CreateRoleWithPolicyOutput - Output of the CreateRoleWithPolicy func
type CreateRoleWithPolicyOutput struct {
	RoleName   string
	RoleArn    string
	PolicyName string
	PolicyArn  string
}

// CreateRoleInput - Output of the CreateRoleWithPolicy func
type CreateRoleInput struct {
	RoleName   string
	RoleArn    string
	PolicyName string
	PolicyArn  string
}

// CreatePrincipalRole - Create a Role, and attach a policy to it
func (am *AccountManager) CreatePrincipalRole() error {

	return nil
}

func (am *AccountManager) createRole() error {
	iamSvc := iam.New(am.awsSession, am.awsConfig)

	return nil
}

// CreateRoleWithPolicy - Create a Role, and attach a policy to it
func (am *AccountManager) CreateRoleWithPolicy(input *CreateRoleWithPolicyInput) (*CreateRoleWithPolicyOutput, error) {

	createRoleRes, err := am.IAM.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(am.principalRoleName),
		AssumeRolePolicyDocument: aws.String(input.AssumeRolePolicyDocument),
		Description:              aws.String(am.principalRoleDescription),
		MaxSessionDuration:       aws.Int64(am.principalRoleMaxSessionDuration),
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
