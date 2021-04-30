# DCE IAM Policies

## Understanding Principal Policies

When an AWS account is added to the `DCE account pool <concepts.html#account-pool>`_, an IAM role and policy are created within the account. This role is assumed by end-users when accessing their leased account.

The principal user's IAM role is returned as `principalRoleArn` when [creating a new account via the DCE API](swagger_link.html). For example:

```json
{
  "id": "123456789012",
  "adminRoleArn": "arn:aws:iam::123456789012:role/OrganizationAccountAccessRole",
  "principalRoleArn": "arn:aws:iam::123456789012:role/DCEPrincipal"
}
```

By default, the `DCEPrincipal` role has _near-administrative_ access to their leased account, with a few exceptions:

- Users may not create AWS support tickets 
    - e.g., we don't want users increasing service limits
- Users may not modify resources required by DCE to manage the child account
    - e.g. users cannot modify the IAM Trust Relationship which allows DCE master to assume into the child account's IAM roles
- Users are limited to a set of configured regions
    - This is to limit the scope of account resets. See `Configuring Account Resets <howto.html#account-resets>`_
- Users are limited to AWS services which DCE knows how to destroy.
    - This is to prevent orphan resources in accounts after reset.

## Principal Role Security

By default, **principal users may elevate their own IAM access**. For example, users may create a new IAM role with an attached `AdministrativeAccess` policy, assign the role to an EC2 instance, and then SSH into the instance as an admin user.

The best way to block this backdoor access to IAM policy elevation is through a [Service Control Policy](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_scp.html), or SCP. An SCP is an organization-level policy which allows administrators to control access to _all_ IAM roles and users within the organizations. 

See `DCE Service Control Policies <#dce-service-control-policies-scp>`_

## DCE Service Control Policies (SCP)

Implementing DCE in an AWS Organization provides the ability to use SCPs, which can be helpful for ensuring the resilience of DCE internal resources. The following SCP is an example policy that contains two statements for protecting your DCE accounts:

- **DenyChangesToAdminPrincipalRoleAndPolicy** is designed to prevent anyone other than the AdminRole from modifying the roles and policies used by DCE.
- **DenyUnsupportedServices** is designed to allow access only to services that are supported by AWS Nuke


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
                "comprehend:*",
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
                "globalaccelerator:*",
                "glue:*",
                "iam:*",
                "imagebuilder:*",
                "iot:*",
                "iotanalytics:*",
                "kafka:*",
                "kinesis:*",
                "kinesisanalytics:*",
                "kinesisvideo:*",
                "kms:*",
                "lakeformation:*",
                "lambda:*",
                "lex:*",
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
                "sts:*",
                "tag:*",
                "transfer:*",
                "waf:*",
                "wafv2:*",
                "waf-regional:*",
                "worklink:*",
                "workspaces:*"
            ],
            "Resource": "*"
        }
    ]
}
```



## Customizing the Principal IAM Policy

Customize the IAM Policies for DCE Principals via Terraform variables. 

See `Configuring Terraform Variables <terraform.html#configuring-terraform-variables>`_.

| Variable | Default | Description |
| --- | --- | --- |
| `principal_policy` | See [principal_policy.tmpl](https://github.com/Optum/dce/blob/master/modules/fixtures/policies/principal_policy.tmpl) | File location for a  IAM principal policy template | 
| `allowed_regions` | _all AWS regions_ | AWS regions which the principal is allowed to access |

The file specified in `principal_policy` is rendered using [golang templates](https://golang.org/pkg/text/template/), and accepts the following arguments:

| Argument | Description |
| --- | --- |
| PrincipalPolicyArn | ARN of the principal IAM policy |
| PrincipalRoleArn | ARN of the principal IAM role |
| AdminRoleArn | ARN of the admin access role within the account |
| PrincipalIAMDenyTags | Populated from the `principal_iam_deny_tags` Terraform variable. By default, these are used to deny access to AWS resources with `AppName=DCE` tags |
| Regions | AWS Regions, populated from the `allowed_regions` Terraform variable |
