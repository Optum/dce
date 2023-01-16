# Local Development

This page will guide you through some basic points for getting started developing against the DCE codebase.

> *Note: unless otherwise noted, all commands shown here should be executed from
> the DCE base directory*

## Configuring your development environment

You may find development easiest on a Mac OS or Linux-based machine. Development should be possible on Windows 10 with the [Windows Subsystem for Linux](https://docs.microsoft.com/en-us/windows/wsl/install-win10) installed, but at the time of this writing has not been verified.

You will need the following:

1. [Go](https://golang.org/doc/install) (version 1.13.x)
1. [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) (version v1.3.x)
1. [GNU make]() (version 3.x)
1. [GNU bash](https://www.gnu.org/software/bash/), which is used for shell scripts
1. An [AWS account](https://aws.amazon.com/premiumsupport/knowledge-center/create-and-activate-aws-account/) for deploying resources
1. The [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) _Note: if you install version 2, see "Configuring AWS CLI 2"_
1. An [AWS IAM user](https://docs.aws.amazon.com/IAM/latest/UserGuide/getting-started_create-admin-group.html) with command line access.
1. [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) (version 2.x)

***Important: deploying DCE to your AWS account can incur cost.***

### Configuring AWS CLI 2

The AWS CLI version 2 includes a breaking change that creates problems with the automation scripts. See
https://docs.aws.amazon.com/cli/latest/userguide/cliv2-migration.html for more information.

For DCE, the recommeonded solution to this problem is to add the following line
in your `~/aws/config` file:

```ini
[default]
cli_pager=
```

### Getting the code locally

To get the code locally fork this repo (https://github.com/Optum/dce) and then clone the repository:

```
git clone https://github.com/${mygithubid}/dce.git dce # replace with your fork's HTTPS URL
cd dce
make setup
```

The last command, `make setup`, will run the `scripts/install_ci.sh` script which will install
the necessary tools for building and testing the project.

## Code Structure

The DCE codebase is comprised of [Go](https://golang.org/) application code, along with [Terraform](https://terraform.io) infrastructure configuration.

The Go code is primarily located within:

- [/cmd](https://github.com/Optum/dce/tree/master/cmd): entrypoint for applications targeting AWS Lambdas and CodeBuild
- [/pkg](https://github.com/Optum/dce/tree/master/pkg): common services used by entrypoint code.

Each subdirectory within the [/cmd/lambda](https://github.com/Optum/dce/tree/master/cmd/lambda) directory targets an individual Lambda function of the same name.


## Building application code

To compile the Go application code, run:

```bash
make build
```

This generates a `/bin/build_artifacts.zip` file, which includes Go binaries for each entrypoint application.

## Unit Tests

Unit tests are located within the `/cmd` and `/pkg` directories, adjacent to their corresponding Go code. So, for example, the code in `/pkg/api/user_test.go` includes tests against `/pkg/api/user.go`.

Execute unit tests by running:

```bash
make test
```

## Code Linting

When you run `make test`, the `lint` target is executed automatically. You can, however, run
the linting by itself by using the command:

```bash
make lint
```

During `make lint`, the `scripts/lint.sh` script executes [golangci-lint](https://github.com/golangci/golangci-lint). The configuration file is `.golangci.yml`. Enabled linters and
rule exceptions can be found in this file.

The `make lint` target also executes [tflint](https://github.com/terraform-linters/tflint)
to lint the [terraform](https://www.terraform.io/) code found in `modules`.

## Functional Tests

Functional tests are located in `tests` and are used to test the integration between a number of services or verify that end-to-end behavior is working properly. For example, we rely heavily on functional tests for DynamoDB interactions, to verify that we are using the DynamoDB SDKs correctly.

Before running functional tests, DCE must be deployed to a test AWS account. **Functional tests truncate the database tables, so do not run them against production environments.**

To deploy DCE for testing, first [authenticate against an AWS test account](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html), then run:

```bash
# Deploy AWS infrastructure using Terraform
cd modules
terraform init
terraform apply

# Deploy application code to AWS
cd ..
make deploy
```

See `Deploying DCE With Terraform <terraform.html#deploy-with-terraform>`_ for more details.

To run functional tests:

```bash
make test_functional
```

Functional tests load the details of the DCE deployment from Terraform module outputs, so there is no need for additional configuration to run functional tests.

## Before committing code

The `make test` target is used by continuous integration build. A failure of the target will
cause the build to fail, so before committing code or creating a pull request you should
run the following commands:

```bash
make build
make test
```


## Building the documentation

Directions coming soon.