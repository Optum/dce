## _next_
- Replace all occurrences of `redbox` to `dce`

## v0.17.0

- Deprecate Launchpad from here
- Modify budget lambdas to write to caching db

## v0.16.0

- Added dynamodb usage_cache
- Fix an issue where the LastModifiedOn property was getting set to a string

## v0.15.1

- Fixed an issue where the IAM policy wasn't being pulled from the module

## v0.15.0

- Added variable for specifying an IAM policy template (GO Template)
- Update IAM Policy for the principal every time the account is unlocked

## v0.14.0

- Added rds backup delete to nuke
- Added Athena resources reset
- Bugfix: In populate_reset_queue lambda, change status from ResetFinanceLock to Active

## v0.13.0

- **BREAKING** Remove Optum-specific rules from the default aws-nuke config.
- **BREAKING** Disable `aws-nuke` by default.  
- Add outputs for DynDB table ARNs

**v0.13.0 Migration Notes**

This release removes a number of Optum-specific configurations from the default aws-nuke YAML configuration. If you want to keep these configurations in your implementation of Redbox, you will need to specify an _override_ nuke config as part of Optum's deployment of Redbox.

To specify a override nuke config, upload your own YAML file to an S3 bucket, and specify the S3 location using the `reset_nuke_template_bucket` and `reset_nuke_template_key` Terraform variables.

This release also disables `aws-nuke` by default, to prevent accidental destruction of critical AWS account resources. To re-enable `aws-nuke`, set the `reset_nuke_toggle` Terraform variable to `"true"`. 

See [README.md for details](./README.md#configuring-aws-nuke) on aws-nuke configuration


## v0.12.3

- Added EKS services to allowed services in policy file, redboxprincipal.go
- Audited alarms and Added API gateway 4XX alarm
- Adds a metadata property to the account object
- Added publish_locks lambda
- Adds a metadata property to the account object


## v0.12.2

- Tag issue, updating to 0.12.2
- Updates nuke whitelist to preserve beta user role policy attachments.

## v0.12.0

- Add SES terraform to enable email from Redbox App for notifications.
- Add budget fields to /leases endpoints
- Cost Explorer spend aggregation service
- Set -up SES Notification for budgets
- Handle "lease-locked" / "lease-unlock" SNS, to add/remove user from AD group
- Budget Checker lambda
- Add `GET /leases` endpoint

## v0.11.1

- Add budget fields to API `/leases` endpoint

- Remove `RedboxAccountAssignment` DyanmoDB table
  - This table was deprecated in v0.10.0, and no longer referenced in AWS Redbox code
- Add `lease-locked` and `lease-unlocked` SNS topics
  - _NOTE:_ No messages are currently being published to these topics. We are supplying them now in advance of further implementation work, so that consumers can start on integration work.

## v0.11.0

- **BREAKING** Add **required** budget fields to API `/leases` endpoint


- Add local functional testing deployment method via Makefile
  - Target "make deploy_local" utilizes scripts/deploy_local terraform to build S3 backend
  - Target "make destroy_local" utilizes scripts/deploy_local terraform and modules/ terraform to destroy environment
- Add LeaseStatusModifiedOn field to Leases DB table
  - includes migration script to add field to existing DB records - scripts/addLeaseStatusModifiedOn
- Fix failed reset builds, caused by failing to assume the accounts `adminRoleArn`
- Fix nuke config, to properly remove policy attachments

## v0.10.0

**BREAKING**

- Rename `Principal` --> `User`; `Assignment` --> `Lease`. Includes:
  - Create new `RedboxLease` table, migrate data from `RedboxAccountAssignment` table
    - Note that in this release, both tables exist in order to allow for migrations. The `RedboxAccountAssignment` is deprecated, and will destroyed in a subsequent release.
  - Rename lamdba functions
  - Refactor code to use new terminology
  - Update API `/leases` endpoints, to use `"principalId"` instead of `"userId"`

## v0.9.2

- Do not nuke AWS*{{id}}\Service or AWS*{{id}}\Read Roles during reset
- Fix CloudWatch alarms to notify after a single failed CodeBuild/Lambda execution

## v0.9.1

- Add terraform outputs for "account-created" and "account-deleted" SNS topics

## v0.9.0

- Add a `DELETE /accounts/{id}` endpoint
  - Removes account from account pool
  - Publishes _account-deleted_ SNS message
  - Delete the IAM Role for the Redbox principal
  - Queues account for reset
- Add `POST /accounts` endpoint"
  - Adds accounts to account pool
  - Publishes _account-created_ SNS message
  - Creates an IAM Role for the Redbox Principal
  - Queues account for reset (initial cleanup)
- Update nuke implementation for `cmd/codebuild/reset`
  - Add functionality to pull a configuration yaml file from an S3 Bucket Object to use for nuke.
  - Add filters for the Account's Admin and User Role in nuke.
  - Rename environment variables used for for nuke.
- Update `Storager` interface and S3 implementation
  - Add `Download` function for downloading an S3 Bucket Object for locally
  - Add acceptance tests for S3 implementation

## v0.8.0

- Updated scripts/migration/v0.7.0_remove_git group_id.go to remove Limit to Update clause
- Add CloudWatch alarm for reset failures (CodeBuild)
- Add `GET /accounts` resource and lambda
- Add `Get /accounts/{id}` endpoint to above

## v0.7.0

- Remove use of "GroupId" in DynamoDB and lambdas/testing.
  - Added "db_group_migrate.go script in scripts directory for one-time use to remove GroupId data from dynamodb
- Modified documentation for database restore functionality in README and script doc

## v0.6.0

- Add new script `scripts/restore_db.sh`
  - Creates a DynamoDB Table from a restore on an existing Backup.
- Updates package name to match public repo.

## v0.5.0

- Add an SNS Topic to allow Provisioning and Decommission messages to be
  published to and be consumed by implementers.
- acctmgr
  - Remove usage of JWT, the body of the request should
    contain the `principalId` in json form that is used as the requestor's
    `PrincipalID`.
  - Sends the create/updated Assignment to the respecitve SNS
    Topic.
  - Response messages are in JSON form. If the request was
    successful, the assignment returns, if it fails, it returns a structured
    Error Response back.
  - No longer returns an error if any with the API Gateway Proxy Response,
    to avoid returning the default "Internal server error" when a lambda returns
    an error. Defaults to returning a nil value for the error.
- Create `pkg/response/error.go` to contain structured Error Responses to be
  used with APIs.
- Update `Notification.go` implementation for `Notificationer` interface.
- Remove `pkg/authorization` and `pkg/common/jwt*` as they are no longer
  integrated with the base aws_redbox.
- Add swaggerUI to host API gateway spec in /dist directory, serving via github pages /docs directory

## v0.4.0

- Added Cloudwatch Alarms to terraform created infrastructure
- Add tflint to `make lint` command
- Add IAM authentication in front of API Gateway endpoints
- Add IAM policy for accessing API Gateway endpoints

## v0.3.0

- Upgrade to Terraform v0.12.0
  - Ran `terraform 0.12upgrade` on terraform code to update to new syntax.
- Upgrade `go.mod`'s terratest to `v0.15.13` to support Terraform v0.12.0.
- Update `tests/acceptance/*.go` tests to use `terraform-0.12.0` as the
  Terraform Binary,

## v0.2.2

- Update `data "template_file" "aws_redbox_api_swagger"` to use the relative
  path of the Terraform module to read the `swaggerRedbox.yaml` file.

## v0.2.1

- Fix JWT parsing for add/remove user APIs

## v0.2.0

- Split apart open-source aws_redbox from Optum-specific implementation:
  - CI/CD pipeline deploys GH release w/artifacts, instead of deploying to AWS
  - Move build scripts into .sh files and make commands, for easier reuse
  - Remove sensitive info from code (account IDs, Launchpad API URLs)
- Remove CodePipeline from reset process (Lambda invokes CodeBuild directly)

## v0.1.0

Initial release
