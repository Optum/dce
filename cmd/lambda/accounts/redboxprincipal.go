package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (c createController) createPrincipalRole(account db.RedboxAccount) (*rolemanager.CreateRoleWithPolicyOutput, error) {
	// Create an assume role policy,
	// to let principals from the same account assume the role.
	//
	// Consumers of open source redbox may modify and customize
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

	// Render the default policy for the Redbox principal
	policyName := c.PrincipalPolicyName
	policy, err := redboxPrincipalPolicy(redboxPrincipalPolicyInput{
		PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", account.ID, policyName),
		PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", account.ID, c.PrincipalRoleName),
		PrincipalIAMDenyTags: c.PrincipalIAMDenyTags,
		AdminRoleArn:         account.AdminRoleArn,
	})
	if err != nil {
		return nil, err
	}

	// Assume role into the new Redbox account
	accountSession, err := c.TokenService.NewSession(&c.AWSSession, account.AdminRoleArn)
	if err != nil {
		return nil, err
	}
	iamClient := iam.New(accountSession)

	// Create the Role + Policy
	c.RoleManager.SetIAMClient(iamClient)
	return c.RoleManager.CreateRoleWithPolicy(&rolemanager.CreateRoleWithPolicyInput{
		RoleName:                 c.PrincipalRoleName,
		RoleDescription:          "Role to be assumed by principal users of Redbox",
		AssumeRolePolicyDocument: assumeRolePolicy,
		MaxSessionDuration:       c.PrincipalMaxSessionDuration,
		PolicyName:               policyName,
		PolicyDocument:           policy,
		PolicyDescription:        "Policy for principal users of Redbox",
		Tags: append(c.Tags,
			&iam.Tag{Key: aws.String("Name"), Value: aws.String("RedboxPrincipal")},
		),
		IgnoreAlreadyExistsErrors: true,
	})
}

var policyTemplate *template.Template

type redboxPrincipalPolicyInput struct {
	PrincipalPolicyArn   string
	PrincipalRoleArn     string
	PrincipalIAMDenyTags []string
	AdminRoleArn         string
}

func redboxPrincipalPolicy(input redboxPrincipalPolicyInput) (string, error) {
	// Compile a template for the policy
	if policyTemplate == nil {
		sess := session.New()
		s3Client := s3.New(sess, aws.NewConfig().WithRegion(os.Getenv("AWS_CURRENT_REGION")))

		bucket := os.Getenv("ARTIFACTS_BUCKET")
		key := "fixtures/policies/redbox_principal_policy.tmpl"
		buff, err := downloadS3Data(s3Client, bucket, key)

		tmpl := template.New("redboxPrincipalPolicy")

		// Add `StringsJoin` function to template
		// See https://stackoverflow.com/a/42724991
		tmpl = tmpl.Funcs(template.FuncMap{"StringsJoin": strings.Join})

		templParsed, err := tmpl.Parse(string(buff))
		if err != nil {
			return "", err
		}
		policyTemplate = templParsed
	}

	// Render template
	buf := &bytes.Buffer{}
	err := policyTemplate.Execute(buf, input)

	return strings.TrimSpace(buf.String()), err
}
