# Policies and Permissions

## Principal Role Policy

Users access their leased accounts through an assumed role. This role also restricts their privileges within their leased account.  The policy is defined [here](https://github.com/Optum/dce/blob/master/modules/fixtures/policies/principal_policy.tmpl).  This policy is designed to protect the IAM principal policy and trusts so that DCE can continue to manage the account.  Additionaly the policy is designed around services that AWS Nuke supports.

## Organizations and Service Control Policies (SCPs)

Implementing DCE in an AWS Organization provides the ability to use SCPs, which can be helpful for ensuring the resilience of your DCE resources. The following SCP is an example policy that contains two statements for protecting your DCE accounts:

- **DenyChangesToAdminPrincipalRoleAndPolicy** is designed to prevent anyone other than the AdminRole from modifying the roles and policies used by DCE.
- **DenyUnsupportedServices** is designed to allow access to services that are supported by AWS Nuke

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
                "iam:ListRoleTags",
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
            "Sid": "DenyUnsupportedServices",
            "Effect": "Deny",
            "NotAction": [
                "acm:*",
                "acm-pca:*",
                "apigateway:*",
                "application-autoscaling:*",
                "appstream:*",
                "athena:*",
                "autoscaling:*",
                "backup:*",
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
                "execute-api:*",
                "firehose:*",
                "fsx:*",
                "glue:*",
                "iam:*",
                "iot:*",
                "kafka:*",
                "kinesis:*",
                "kinesisanalytics:*",
                "kinesisvideo:*",
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
                "opsworks-cm:*",
                "rds:*",
                "redshift:*",
                "rekognition:*",
                "resource-groups:*",
                "robomaker:*",
                "route53:*",
                "s3:*",
                "sagemaker:*",
                "secretsmanager:*",
                "servicecatalog:*",
                "servicediscovery:*",
                "ses:*",
                "sns:*",
                "sqs:*",
                "ssm:*",
                "states:*",
                "storagegateway:*",
                "sts:*",
                "tag:*",
                "waf:*",
                "waf-regional:*",
                "worklink:*",
                "workspaces:*"
            ],
            "Resource": "*"
        }
    ]
}
```
