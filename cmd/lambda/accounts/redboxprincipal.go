package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
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
		tmpl := template.New("redboxPrincipalPolicy")

		// Add `StringsJoin` function to template
		// See https://stackoverflow.com/a/42724991
		tmpl = tmpl.Funcs(template.FuncMap{"StringsJoin": strings.Join})

		tmpl, err := tmpl.Parse(`
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Sid": "DoNotModifySelf",
        "Effect": "Deny",
        "NotAction": [
          "iam:GetPolicy",
          "iam:GetPolicyVersion",
          "iam:GetRole",
          "iam:GetRolePolicy",
          "iam:ListRoles",
          "iam:ListRolePolicies",
          "iam:ListAttachedRolePolicies",
          "iam:ListRoleTags",
          "iam:ListPoliciesGrantingServiceAccess",
          "iam:ListEntitiesForPolicy",
          "iam:ListPolicyVersions",
          "iam:GenerateServiceLastAccessedDetails"
        ],
        "Resource": [
          "{{.PrincipalPolicyArn}}",
          "{{.PrincipalRoleArn}}",
          "{{.AdminRoleArn}}" 
        ]
      },
      {
        "Sid": "DenyTaggedResourcesAWS",
        "Effect": "Deny",
        "Action": "*",
        "Resource": "*",
        "Condition": {
          "StringEquals": {
            "aws:ResourceTag/AppName": [
              "{{ StringsJoin .PrincipalIAMDenyTags "\", \"" }}"
            ]
          }
        }
      },
      {
        "Sid": "DenyIAM",
        "Effect": "Deny",
        "Action": [
          "iam:DeactivateMFADevice",
          "iam:CreateGroup",
          "iam:DeleteGroup",
          "iam:UpdateUser",
          "iam:UpdateGroup",
          "iam:CreateRoleWithPolicy",
          "iam:DeleteUser",
          "iam:CreateSAMLProvider",
          "iam:CreateUser",
          "iam:RemoveUserFromGroup",
          "iam:AddUserToGroup",
          "iam:UpdateAccountPasswordPolicy",
          "iam:DeleteVirtualMFADevice",
          "iam:EnableMFADevice",
          "iam:CreateAccountAlias",
          "iam:DeleteAccountAlias",
          "iam:UpdateSAMLProvider",
          "iam:ChangePassword",
          "iam:DeleteSAMLProvider"
        ],
        "Resource": "*"
      },
      {
        "Sid": "ViewBillingAndBudgets",
        "Effect": "Allow",
        "Action": [
          "aws-portal:ViewBilling",
          "aws-portal:ViewUsage",
          "budgets:ViewBudget"
        ],
        "Resource": "*"
      },
      {
        "Sid": "AllowedServices",
        "Effect": "Allow",
        "Action": [
          "acm:*",
          "apigateway:*",
          "appstream:*",
          "autoscaling:*",
          "batch:*",
          "cloud9:*",
          "clouddirectory:*",
          "cloudformation:*",
          "cloudfront:*",
          "cloudhsm:*",
          "cloudsearch:*",
          "cloudtrail:*",
          "cloudwatch:*",
          "codebuild:*",
          "codecommit:*", 
          "codedeploy:*", 
          "codepipeline:*",
          "codestar:*",
          "cognito-identity:*",
          "cognito-idp:*",
          "cognito-sync:*",
          "config:*",
          "datapipeline:*",
          "devicefarm:*",
          "dynamodb:*",
          "ec2:*",
          "ecs:*",
          "elasticfilesystem:*",
          "elasticloadbalancing:*",
          "elasticmapreduce:*",
          "elasticbeanstalk:*",
          "elastictranscoder:*",
          "elasticache:*",
          "eks:*",
          "fsx:*",
          "firehose:*",
          "glue:*",
          "iam:*",
          "iot:*",
          "kms:*",
          "kinesis:*",
          "lambda:*",
          "mq:*",
          "machinelearning:*",
          "mediaconvert:*",
          "mediapackage:*",
          "mediastore:*",
          "opsworks:*",
          "rds:*",
          "redshift:*",
          "rekognition:*",
          "resource-groups:*",
          "route53:*",
          "s3:*",
          "sns:*",
          "sqs:*",
          "ssm:*",
          "sagemaker:*",
          "secretsmanager:*",
          "servicecatalog:*",
          "storagegateway:*",
          "waf:*",
          "waf-regional:*",
          "workspaces:*",
          "a4b:*",
          "comprehend:*"
        ],
        "Resource": "*",
        "Condition": {
          "StringEquals": {
            "aws:RequestedRegion": [
              "us-east-1",
              "us-west-1"
            ]
          }
        }
      }
    ]
  }
	`)
		if err != nil {
			return "", err
		}
		policyTemplate = tmpl
	}

	// Render template
	buf := &bytes.Buffer{}
	err := policyTemplate.Execute(buf, input)

	return strings.TrimSpace(buf.String()), err
}
