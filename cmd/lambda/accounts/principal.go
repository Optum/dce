package main

import (
	"fmt"
	"strings"

	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

func (c createController) createPrincipalRole(account db.Account) (*rolemanager.CreateRoleWithPolicyOutput, string, error) {
	// Create an assume role policy,
	// to let principals from the same account assume the role.
	//
	// Implementors of DCE may modify and customize
	// this as need (eg. to integrate with SSO/SAML)
	// by responding to the "account-created" SNS topic
	assumeRolePolicy := strings.TrimSpace(fmt.Sprintf(`
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
	`, account.ID))

	// Render the default policy for the principal role
	policyName := c.PrincipalPolicyName
	policy, policyHash, err := c.StoragerService.GetTemplateObject(c.ArtifactsBucket, c.PrincipalPolicyS3Key,
		principalPolicyInput{
			PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", account.ID, policyName),
			PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", account.ID, c.PrincipalRoleName),
			PrincipalIAMDenyTags: c.PrincipalIAMDenyTags,
			AdminRoleArn:         account.AdminRoleArn,
		})
	if err != nil {
		return nil, "", err
	}

	// Assume role into the new child account
	accountSession, err := c.TokenService.NewSession(&c.AWSSession, account.AdminRoleArn)
	if err != nil {
		return nil, "", err
	}
	iamClient := iam.New(accountSession)

	// Create the Role + Policy
	c.RoleManager.SetIAMClient(iamClient)
	createRoleOutput := &rolemanager.CreateRoleWithPolicyOutput{}
	createRoleOutput, err = c.RoleManager.CreateRoleWithPolicy(&rolemanager.CreateRoleWithPolicyInput{
		RoleName:                 c.PrincipalRoleName,
		RoleDescription:          "Role to be assumed by principal users",
		AssumeRolePolicyDocument: assumeRolePolicy,
		MaxSessionDuration:       c.PrincipalMaxSessionDuration,
		PolicyName:               policyName,
		PolicyDocument:           policy,
		PolicyDescription:        "Policy for principal users",
		Tags: append(c.Tags,
			&iam.Tag{Key: aws.String("Name"), Value: &c.PrincipalRoleName},
		),
		IgnoreAlreadyExistsErrors: true,
	})
	return createRoleOutput, policyHash, err
}

type principalPolicyInput struct {
	PrincipalPolicyArn   string
	PrincipalRoleArn     string
	PrincipalIAMDenyTags []string
	AdminRoleArn         string
}
