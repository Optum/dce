# Local Development

This page will guide you through some basic points for getting started developing against the DCE codebase.

## Code Structure

The DCE codebase is comprised of [Go](https://golang.org/) application code, along with [Terraform](https://terraform.io) infrastructure configuration.

The Go code is primarily located within:

- [`/cmd`](https://github.com/Optum/dce/tree/master/cmd): entrypoint for applications targeting AWS Lambdas and CodeBuild
- [`/pkg`](https://github.com/Optum/dce/tree/master/pkg): common services used by entrypoint code.

Each subdirectory within the [`/cmd/lambda`](https://github.com/Optum/dce/tree/master/cmd/lambda) directory targets an individual Lambda function of the same name.

## Building application code

To compile the Go application code, run:

```
make build
```

This generates a `/bin/build_artifacts.zip` file, which includes Go binaries for each entrypoint application.

## Unit Tests

Unit tests are located within the `/cmd` and `/pkg` directories, adjacent to their corresponding Go code. So, for example, the code in `/pkg/api/user_test.go` includes tests against `/pkg/api/user.go`.

Execute unit tests by running:

```
make test
``` 

## Functional Tests

Functional tests are used where we want to test the integration between a number of services or verify that end-to-end behavior is working properly. For example, we rely heavily on functional tests for DynamoDB interactions, to verify that we are using the DynamoDB SDKs correctly.

Before running functional tests, DCE must be deployed to a test AWS account. **Functional tests truncate the database tables, so do not run them against production environments.**

To deploy DCE for testing, first [authenticate against an AWS test account](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html), then run:

```bash
# Deploy AWS infrastructure using Terraform
cd modules
terraform init
terraform apply

# Deploy application code to AWS
make deploy 
``` 

See `Deploying DCE With Terraform <terraform.html#deploy-with-terraform>`_ for more details.

To run functional tests:

```bash
make test_functional
```

Functional tests load the details of the DCE deployment from Terraform module outputs, so there is no need for additional configuration to run functional tests.