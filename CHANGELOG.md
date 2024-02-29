## v0.36.1

- Fix: Remove default 25 scan limit when querying for leases

## v0.36.0

- change aws-nuke to directly use opensource version

## v0.35.3

- upgrade the aws nuke version from  v2.0.2 to v2.0.3

## v0.35.2

- upgrade the aws nuke version from  v2.0. 1 to v2.0.2

## v0.35.1

- Fix: Terraform formatting

## v0.35.0

- Add: Added blanket * allow permission in place of specific allow permissions for easier integration with future AWS products github.com/Optum/dce (#467)

## v0.34.2

- Add: Added network-firewall:* permission in place of List and describe in openSource repo github.com/Optum/dce (#455)

## v0.34.1

- Add: Enable network-firewall:ListRuleGroups permission (#449)
- Fix: Recent changes to AWS S3 default bucket policy have broken the install. Removing ACL from the bucket allows the installation to proceed.

## v0.34.0

- Fix: get latest aws-nuke release. (#432)

## v0.33.9

- Add AWS Backup MountCapsule permission [#426](https://github.com/Optum/dce/issues/426)
- Add new secondary index 'PrincipalIdLastModifiedOn' for Lease table with range key as LastModifiedOn to get the records sort by last-modified
- Update pkg/data/leases.go queryLeases method to use new IndexName PrincipalIdLastModifiedOn instead of existing IndexName PrincipalId in to get leases in order
- Fix extra characters being added on inputs passed to deployment script in pipeline [#433](https://github.com/Optum/dce/issues/433)
- Encrypt SNS topics by default
- Upgrade to Go version 1.17
- Upgrade Ubuntu version on Azure DevOps Agent
- Fix Go dependency errors in pipeline
- Require at least `1.3.x` or later versions of Terraform to recieve new provider updates
- Output `codebuild_reset_role_name` so you can add a policy to it to grant access to `reset_nuke_template_bucket`.

## v0.33.8

- Upgrade the Swagger UI dependency to remove a very dangerous vulnerability (upgrade Swagger UI to v3.51.2).

## v0.33.7

- Update requirements packages for docs.
- Update gotool version for pipeline.

## v0.33.6

- Add LakeFormation to the principal IAM policy.
- Update mocks
- updates requirements packages for docs

## v0.33.5

- rename pipeline credentials inputs

## v0.33.4

- Upgrade aws provider to v2.65.0 to fix SES configuration set. [terraform-provider-aws #12024](https://github.com/hashicorp/terraform-provider-aws/pull/12024)
- Add firewall manager to principal IAM policy.

## v0.33.3

- Remove apigatewayv2 from the principal IAM policy.
- Update to the principal IAM policy name lex-models.

## v0.33.2

- Add to the principal IAM policy:
  - apigatewayv2
  - comprehend
  - globalaccelerator
  - imagebuilder
  - lex
  - transfer
  - wafv2

## v0.33.1

- Fix populate reset queue when dynamodb returns paginated result
- Add account status to last evaluated key when querying account table using global secondary index
- Fix gosec issue related to G601 (CWE-118): Implicit memory aliasing in for loop

## v0.33.0

- Upgrade aws-nuke to v1.3.0

## v0.32.0

- Upgrade go to 1.15
- Upgrade go mod dependencies version.
- Improved test scenarios.
- Add `iotanalytics:*` policy to the set of allowed services.
- Increase timeout for functional test execution.

## v0.31.3

- Fix data and update visualization for codebuild reset widget and error scrappers.

## v0.31.2

- Fix bug: Status change in account table fails for leased accounts that are expired. See <https://github.com/Optum/dce/issues/344>
- Fix bug: When lease starts today, fixed usage start and end date that was printed in log message.

## v0.30.1

- Added new tool in `tools` folder for generating Markdown and IAM example policy for AWS Nuke
  support. See README in `tools/awsnukedocgen`.
- Added new services supported by DCE: Kinesis Analytics, Kinesis Video, Opsworks CM, Robomaker,
  and Worklink.
- Added new `make setup` target for installing tools on local developer machines.
- Hardened `scripts/install_ci.sh` to be used by developers.
- Added documenation for local developer environment setup.
- Fix bug where update lease should update LeaseStatusModifiedOn
- Fix bug with delete lease, replace list method call with GetByAccountIdAndPrincipalId.
- Fix to issue to include tflint package for MacOS user and upgrade version for Linux user.
- Fixed a bug causing admins to be authorized as users.

## v0.29.0

- Add `tag:*` to support finding resources by tag, and changing tags for existing resources.
- add `application-autoscaling:*` to support auto-scaling targets and policies
- Fix bug where account status would not transition from `Leased` --> `NotReady`, on lease deletion.
- Do not run `aws-nuke` if `var.reset_nuke_toggle = false`. Previously, ran aws-nuke in dry-run mode.
- Remove Lease table stream processing to change the account status to be `NotReady`
- Add Account Service to the Lease Service so we can reset account status directly when removing or adding a lease
- Update Populate Reset Queue to skip the Account Service and publish the Reset event directly
- Update the visibility timeout on the Reset SQS to be 6 times the runtime of the process reset queue Lambda
- Fix an issue when comparing the current Principal Policy Hash to the new version where the pointers were being compared and not the values
- Update functional testing timeout to be 20 minutes from the default of 10 minutes
- Fix to allow budget notification email templates > 4Kb
- Prevent non-admins from performing CRUD operations on other user's leases

## v0.28.0

- Add `usage_ttl` Terraform var, to configure Usage DB record TTLs.
- Added account pool status monitoring and dashboard widget
- Allow `athena:*` for DCE Principal IAM role

## v0.27.0

- Add CloudWatch Dashboard for monitoring DCE
- Support deleting a lease by ID at `GET /leases/{ID}` endpoint
- Update `GET /accounts` to allow for querying with `adminRoleArn`, `principalRoleArn`, and `principalPolicyHash`
- Add `sts:*` to principal IAM policy, and to documented SCP

## v0.26.0

- **BREAKING CHANGE** Change `GET /leases` to always return a list
- **BREAKING CHANGE** Change `GET /leases` to not return 404 when the list is empty

**Migration Notes**

This release makes breaking changes to the `GET /leases` endpoint, so that requests will always return a HTTP 200 response, with a JSON array in the payload, even if the result set is empty. Previously, if a query had no results, the endpoint would return an HTTP 404 response, with an error object in the response body.

DCE API clients will need to be updated accordingly, to handle this response.

## v0.25.0

- **BREAKING CHANGE:** Set the default allowed regions to us-east-1 only
- Support query params for `GET /accounts` endpoint
- Fixed bug causing dce auth web page to fail
- Fix incorrect `POST /leases` validation errors on principal budget (#214)
- Fix missing regions config from nuke template (#221)
- Add `execute-api:*` to DCE Principal policy (#224)

**Migration Notes**

This release changes the list of allowed regions to only include `us-east-1` by default. This is in order to reduce the time it takes for account reset CodeBuilds to run. Previously, these codebuilds would take 1h+ to nuke the 18 default regions, even on an empty account.

The list of allowed regions is configurable as an `allowed_regions` Terraform variable, and may be set to any region names supported by AWS.

## v0.24.1

- Fix failure to render IAM principal policy in `update_principal_policy` lambda (#207)

## v0.24.0

- **BREAKING CHANGE:** Enable aws-nuke to run by default (disable dry-run mode)
- Update Status Lambda - budget_check: Terminate lease if spend > Principal budget amount
- Support `metadata` parameter in `/accounts` API endpoints
- Add `PUT /accounts/:id` endpoint
- Fixed bug where child account's DCEPrincipal role trusted itself rather than the master account
- Add `GET /auth` and `GET /auth/{file+}` endpoints for retrieving credentials web page
- Merged quickstarts into how-to guide
- Support query params for `GET /usage` endpoints

**Migration Notes**

In previous versions of DCE, aws-nuke would run in [_Dry Run Mode_](https://github.com/rebuy-de/aws-nuke#usage) by default, during account resets. This version changes that default behavior, so that aws-nuke runs to completion by default.

If you would like to maintain previous behavior and run aws-nuke in _Dry Run Mode_, you may set the terraform variable:

```hcl
reset_nuke_toggle = false
```

See documentation.

## v0.23.0

- Added `/accounts?accountStatus=<status>` URL for querying accounts by status.
- Added Lease Validation for check against max budget amount, max budget period, principal budget amount and principal budget period
- Increase the threshold for Reset CodeBuild alarms to 10 failures over 5 hours.
- Support `metadata` field in `POST /leases` endpoint
- Fix bug where lease expiredOn/budgets/etc. were not being updated, if the account was previously used by the lease principal.

## v0.22.0

**BREAKING CHANGES**

This release includes changes to rename every reference of "Redbox" to "DCE".
In many cases, we removed namespaces entirely: for example, we'll refer to an `account` rather
than a `dceAccount` wherever possible.

This release breaks a number of interfaces, which may require updates to DCE clients.

For example:

- Terraform outputs have been renamed (eg. `redbox_account_db_table_name` is now `accounts_table_name`)
- SNS topics have been renamed (eg `redbox-account-created` is now `account-created`)
- The name of the IAM Principal role and policy have been renamed (`DCEPrincipal` / `DCEPrincipalDefaultPolicy`)

This release also removes the deprecated DynamoDB tables with "Redbox" prefixes.

## v0.21.0

**BREAKING CHANGES**

- Rename DynamoDB tables (does not remove old tables)
  - RedboxAccountProd --> Accounts
  - RedboxLeaseProd --> Leases
  - UsageCache --> Usage

**Migration Notes**

_DynamoDB Migration_

As part of the v0.21.0 release, we are renaming all our DynamoDB tables to remove the "Redbox" prefix, and to standardize naming conventions.

DynamoDB does not support in-place table renaming, so we will need to migrate data from each table to the newly renamed table.

To do this, you may run the migration script in [/scripts/migrations/v0.21.0_rename_db_tables_dce](https://github.com/Optum/dce/blob/master/scripts/migrations/v0.21.0_rename_db_tables_dce/main.go). This script will copy all data from the old tables to the new tables.

Note that this release does **_not_** delete the old tables, to provide the opportunity to migrate data. Subsequent releases _will_ destroy the old tables.

## v0.20.0

- Fixed a bug in a migration script
- Fixed output from publish_lease_events that was generating confusing log entries.
- Cleaned up naming for scheduling the update_lease_status lambda
- Cleaned up naming for scheduling populate_reset_queue lambda to remove
  "weekly" and scheduled the lambda for every six hours instead of weekly.
- Add `POST /leases/:id/auth` script, to generate STS creds for a leased account

## v0.19.2

- Fixed issue with the lease check logic that was expiring non-expired leases.
- Migration script to fix wrongly expired leases

## v0.19.1

- Fixed issue with lease status reason not being set when the lease was newly created.

## v0.19.0

**BREAKING CHANGES**

- Add unique ID to Leases DB and API records
- Move to an _Expiring Leases model_ (see below for details)

_Other Changes_

- Add ECR to DCE user principal policy
- Add email with attachment
- Added expiration date for lease ends
- Lease added SNS topic updates principal policy
- Refactored lease API controller and methods to organize methods into files.
- Add functions to evaluate who is calling an API and what their role is

### Migration Notes for v0.19.0

In order to upgrade your DCE deployment to v0.19.0, you will need to:

- Run the migration script located in `scripts/migrations/v0.19.0_db_expiring_leases`
  - Adds a new `id` field to all existing `Lease` records
  - Sets a default expiration date for all existing `Lease` records
    - **IMPORTANT** you must override [the default expiration date](https://github.com/Optum/dce/blob/master/scripts/migrations/v0.19.0_db_expiring_leases/main.go#L65)
  - Marks all `*Locked` leases as `Inactive`
- Update any DCE API clients to include the `expiresOn` property in their `Lease` record.

### _Expiring Leases Model_

Prior to v0.19.0, leases were held in perpetuity by principals, or until the principal removed their lease via the `DELETE /leases` endpoint. Leased accounts would be "reset" at the end of the week. During reset, the lease would be marked as _Locked_, and then marked as _Active_ again after the reset was complete.

As of v0.19.0, leases are held for a defined time period (defined by the `expiresOn` property), and then destroyed (marked as `Inactive`). Accounts are reset after the leases expires. There is no longer any type of `*Locked` state, as leases are always either `Active` or `Inactive`.

Changes for this new behavior include:

- Simplified lease status model to include only two statuses: Inactive and Active.
- Changed check_budget to update_lease_status and added check for expiration date.
- Changed SQS and SNS notifications for lease status change to be triggered by lease status change in DB.
- Added <https://readthedocs.org/> style documentation, `make documentation` target
- Added generation for API documentation from Swagger YAML to <https://readthedocs.org/> format.
- Added defaults for leases; if ID isn't specified upon save in the DB a new one will be assigned, and if
  the expiration date isn't defined the environment variable `DEFAULT_LEASE_LENGTH_IN_DAYS` will be used and
  if that is not defined, a default of seven (7) days will be used.
- Added migration for the leases to all be set to Inactive if they're anything but Active.

## v0.18.1

- Fix IAM policy for DCE principal, to allow full access to CloudWatch logs

## v0.18.0

- Minor fixes to `scripts/deploy_local/deploy_local_build.sh` for options to be recognized.
- README updates to include current steps for build and deployment.
- Pull requests authored by non-team members will not build until a team member comments
- Add usage table arn to tf output
- Adds GET /leases API support

## v0.17.0

- Deprecate Launchpad from here
- Modify budget lambdas to write to caching db
- Add `GET /usage` endpoint, to retrieve usage for leases

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
