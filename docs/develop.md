# Local Development

DCE was born at Optum, but belongs to the community. We eagerly welcome community contributions.

This page will guide you through some basic points for getting started developing against the DCE code base.

## Code Structure

The DCE code base is comprised of [Go](https://golang.org/) application code, along with Terraform infrastructure configuration.

The Go code is primarily located within:

- [`/cmd`](https://github.com/Optum/dce/tree/master/cmd): entrypoint for applications targeting AWS Lambdas and CodeBuild
- [`/pkg`](https://github.com/Optum/dce/tree/master/pkg): common services used by entrypoint code.

Note that each subdirectory within the [`/cmd/lambda`](https://github.com/Optum/dce/tree/master/cmd/lambda) directory targets an individual Lambda function of the same name.

## Building application code

To compile the Go application code, you may run:

```
make build
```

This will generate a `/bin/build_artifacts.zip` file, which includes Go binaries for each entrypoint application.

## Unit Tests

Unit tests are located within the `/cmd` and `/pkg` directories, adjacent to their corresponding Go code. So, for example, the code in `/pkg/api/user_test.go` includes tests against `/pkg/api/user.go`.

You may execute unit tests by running:

```
make test
``` 

## Functional Tests

Functional tests are used where we want to test the integration between a number of services or verify that end-to-end behavior is working properly. For example, we rely heavily on functional tests for DynamoDB interactions, to verify that we are using the DynamoDB SDKs correctly.

Before running functional tests, you will need to deploy DCE to a test account. Note that **functional tests truncate the database tables, so do not run them against production accounts.**

To deploy DCE for testing, first [login to an AWS test account](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html), then run:

```bash
# Deploy AWS infrastructure using Terraform
cd modules
terraform init
terraform apply

# Deploy application code to AWS
make deploy 
``` 

See [_Deploying DCE With Terraform_](terraform.md#deploying-dce-with-terraform) documentation for more details.

You may then run functional test against your deployed instance of DCE:

```bash
make test_functional
```

Note that functional tests load the details of your DCE deployment from your Terraform module outputs, so there is no need for additional configuration to run functional tests.