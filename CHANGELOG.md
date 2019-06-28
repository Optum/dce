## _next_

- Update nuke implementation for `cmd/codebuild/reset` 
  - Add functionality to pull a configuration yaml file from an S3 Bucket Object
  to use for nuke. 
  - Add filters for the Account's Admin and User Role in nuke.
  - Rename environment variables used for for nuke.
- Update Storager interface and S3 implementation
  - Add `Download` function for downloading an S3 Bucket Object for locally
  - Add acceptance tests for S3 implementation
- Adds a `DELETE /accounts/{id}` endpoint
- Add `POST /accounts` endpoint, to add new AWS accounts to the pool

## v0.8.0

- Updated scripts/migration/v0.7.0_remove_group_id.go to remove Limit to Update clause
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
    contain the `userId` in json form that is used as the requestor's
    `UserID`.
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
