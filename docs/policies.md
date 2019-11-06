# Policies and Permissions

## Principal Role Policy

Any user that access their's lease will assume a role in the leased account.  When they assume this role they are restricted to what they access.  The policy is defined [here](https://github.com/Optum/dce/blob/master/modules/fixtures/policies/principal_policy.tmpl).  This policy is designed to protect the IAM principal policy, trusts so that DCE can continue to manage the account, and around services that AWS Nuke supports.

## Organizations and Service Control Policies (SCPs)

It is possibly to implement DCE inside an AWS Organization.  There are additional benefits when doing this type of implementation including the ability to use SCPs.  The following is an example SCP policy that can be implemented to better protect DCE accounts.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "DenyChangesToAdminPrincipalRoleAndPolicy",
            "Effect": "Deny",
            "NotAction": [
                "iam:GetContextKeysForPrincipalPolicy",
                "iam:GetRole",
                "iam:GetRolePolicy",
                "iam:ListAttachedRolePolicies",
                "iam:ListInstanceProfilesForRole",
                "iam:ListRolePolicies",
                "iam:ListRoleTags"
            ],
            "Resource": [
                "arn:aws:iam::*:role/AdminRole",
                "arn:aws:iam::*:role/DCEPrincipal*",
                "arn:aws:iam::*:policy/DCEPrincipal*"
            ],
            "Condition": {
                "StringNotLike": {
                    "aws:PrincipalARN": "arn:aws:iam::*:role/AdminRole"
                }
            }
        },
        {
            "Sid": "DenyIAM",
            "Effect": "Deny",
            "Action": [
            "iam:DeactivateMFADevice",
            "iam:CreateSAMLProvider",
            "iam:UpdateAccountPasswordPolicy",
            "iam:DeleteVirtualMFADevice",
            "iam:EnableMFADevice",
            "iam:CreateAccountAlias",
            "iam:DeleteAccountAlias",
            "iam:UpdateSAMLProvider",
            "iam:DeleteSAMLProvider"
            ],
            "Resource": "*"
        },
        {
            "Sid": "DenyUnsupportedServices",
            "Effect": "Deny",
            "NotAction": [
                "acm:*",
                "acm-pca:*",
                "apigateway:*",
                "appstream:*",
                "autoscaling:*",
                "aws-portal:*",
                "backup:*",
                "batch:*",
                "budgets:*",
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
                "dax:*",
                "devicefarm:*",
                "dms:*",
                "ds:*",
                "dynamodb:*",
                "ec2:*",
                "ecr:*",
                "ecs:*",
                "eks:*",
                "elasticache:*",
                "elasticbeanstalk:*",
                "elasticfilesystem:*",
                "elasticloadbalancing:*",
                "elasticmapreduce:*",
                "elastictranscoder:*",
                "es:*",
                "events:*",
                "firehose:*",
                "fsx:*",
                "glue:*",
                "iam:*",
                "iot:*",
                "kafka:*",
                "kinesis:*",
                "kms:*",
                "lambda:*",
                "lightsail:*",
                "logs:*",
                "machinelearning:*",
                "mediaconvert:*",
                "medialive:*",
                "mediapackage:*",
                "mediastore:*",
                "mediatailor:*",
                "mobilehub:*",
                "mq:*",
                "neptune-db:*",
                "opsworks:*",
                "rds:*",
                "redshift:*",
                "rekognition:*",
                "resource-groups:*",
                "route53:*",
                "s3:*",
                "sagemaker:*",
                "sdb:*",
                "secretsmanager:*",
                "servicecatalog:*",
                "servicediscovery:*",
                "servicequotas:*",
                "ses:*",
                "sns:*",
                "sqs:*",
                "ssm:*",
                "states:*",
                "storagegateway:*",
                "waf-regional:*",
                "waf:*",
                "workspaces:*"
            ],
            "Resource": "*"
        }
    ]
}
```
