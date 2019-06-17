## _next_

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
