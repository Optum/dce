# AWS Redbox

The premier repository for AWS Redbox.
=======


<!--
Generated with markdown-toc
https://github.com/jonschlinkert/markdown-toc
markdown-toc -i README.md --maxdepth 3
-->

<!-- toc -->

- [API Reference](#api-reference)
  * [API Location](#api-location)
  * [Authorization](#authorization)
- [Scripts](#scripts)
  * [`scripts/build.sh`](#scriptsbuildsh)
  * [`scripts/deploy.sh`](#scriptsdeploysh)
- [Build & Deploy](#build--deploy)
- [Lambda Overview](#lambda-overview)
- [CodeBuild Overview](#codebuild-overview)
- [Database Schema](#database-schema)
- [Reset](#reset)
  * [Nuke](#nuke)
  * [Launchpad](#launchpad)
- [Directory Structure](#directory-structure)
- [Account Provisioning & Decommissioning](#account-provisioning--decommissioning)

<!-- tocstop -->

## API Reference

Redbox exposes an API for managing Redbox accounts and assignments.

See [swaggerRedbox.yaml](./modules/swaggerRedbox.yaml) for endpoint documentation.

### API Location

The API is hosted by AWS API Gateway. The base URL is exposed as a Terraform output. To retrieve the base url of the API, run the following command from your Terraform module directory:

```
terraform output api_url
```

### Authorization

The Redbox API is authorized via IAM. To access the API, you must have access to an IAM principal with [appropriate IAM access to execute the API](https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-iam-policy-examples-for-api-execution.html).

All API requests must be signed with Signature Version 4. See [AWS documentation for signing requests](https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html).

#### IAM Policy for Redbox API requests

The IAM principal used to send requests to the Redbox API must have sufficient permissions to execute API requests.

The Terraform module in the repo provides an IAM policy with appropriate permissions for executing Redbox API requests. You can access the policy name and ARN as Terraform outputs.

```
terraform output api_access_policy_name
terraform output api_access_policy_arn
``` 

#### Signing requests in Go

The AWS SDK for Go exposes a [`signer/v4` package](https://docs.aws.amazon.com/sdk-for-go/api/aws/signer/v4/), which may be used to sign API requests. For example:

```go
import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	sigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"net/http"
	"time"
)

func sendRequest(method, endpoint) (http.Response, error) {
	// Create an API request
	req, err := http.NewRequest(method, apiUrl+endpoint, nil)
  if err != nil {
  	return nil, err
  }
	
	// Load credentials from env vars, or a credentials file
	awsCreds := credentials.NewChainCredentials([]credentials.Provider{
    &credentials.EnvProvider{},
    &credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
  })
	
	// Sign the request
	signer := sigv4.NewSigner(awsCreds)
	signedHeaders, err := signer.Sign(req, nil, "execute-api", "us-east-1", time.Now())
	if err != nil {
    return nil, err
  }
	
	// Send the API request
	return http.DefaultClient.Do(req)
}
```

#### Signing requests in Python

See AWS docs with [example code for signing requests in Python](https://docs.aws.amazon.com/general/latest/gr/sigv4-signed-request-examples.html).

Alternatively, you could consider open-source libraries like [aws-requests-auth](https://github.com/DavidMuller/aws-requests-auth) for signing requests.

#### Signing requests in Postman 

See AWS docs for [sending signed requests in Postman](https://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-use-postman-to-call-api.html)

## Scripts

### `scripts/build.sh`

**Assumes that it runs in the root directory of the repo.**

Bash script to unit test all Go projects and build all executables in the
`cmd` directory, and generate the `bin/build_artifacts.zip` and
`bin/terraform_artifacts.zip` containing individual zipped executables and
Terraform files.

Requirements

- [Go v1.12.1](https://golang.org/dl/)
- [golangci-lint v1.16.0](https://github.com/golangci/golangci-lint/releases/tag/v1.16.0)

Artifacts

- `bin/lambda/`
  - Executables and Zips generated from Golang Lambda Functions in
    `cmd/lambda/`. Gets removed at the end.
- `bin/codebuild/`
  - Executables and Zips generated from Golang CodeBuild functions
    in `cmd/codebuild/`. Gets removed at the end.
- `bin/build_artifacts.zip`

  - The Build Artifact Zip file containing all the generated Zips from
    `bin/lambda/` and `bin/codebuild/`. - `lambda` - `acctmgr.zip` - `financelock.zip` - `resetsqs.zip` - `resettrigger.zip` - `codebuild` - `reset.zip`

- `bin/terraform_artifacts.zip`
  - The Terraform Artifact Zip file containing the `modules` directory
    with all of the base Terraform Files.

### `scripts/deploy.sh`

**Assumes that it runs in the root directory of the repo.**

Bash script that finds `bin/build_artifacts.zip`, unzips it, and uploads those artifacts
(CodeBuild pipeline as well as all lambdas) to the designated S3 artifact bucket. It will also
link those lambdas to by running `lambda update-function-code`. You must run `scripts/build.sh`
prior to running `scripts/deploy.sh`.

#### Usage

```bash
~$ ./scripts/deploy.sh <namespace> <artifactBucket>
```

| Argument         | Description                                             |
| ---------------- | ------------------------------------------------------- |
| `namespace`      | Indicates which namespace this deployment is scoped to. |
| `artifactBucket` | Describes which S3 artifact bucket to use.              |

## Build & Deploy

Run `make build` to compile all lambdas under the `functions` directory.
This will produce the binaries (as well as the zips that can be uploaded to the
AWS console for manual deployment) in the `bin` directory. See the `deploy.sh`
above for automated deployment.

## Lambda Overview

Each Lambda has its own directory under `cmd/lambda/{function}` for the
source code itself. The executable will be built under the `bin` directory,
zipped up, and deployed to the S3 artifact bucket. The Lambda Function will
then be deployed and updated via AWS CLI calls.

| Function     | Description                                                                                                                                                                                                    |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| acctmgr      | Functionality for routing requests, adding/removing users to/from the ad group and updating the dynamodb account manifest.                                                                                     |
| resettrigger | Functionality to drain an SQS Queue containing AWS Redbox Accounts to be reset and will trigger each account's Reset CodePipeline respectively. Updates the DynamoDB Account Assignment manifest if necessary. |

## CodeBuild Overview

Each CodeBuild function has its own directory under
`cmd/codebuild/{function}` for the source code itself. The executable
will be built under the `bin` directory, zipped up, and deployed to the S3
artifact bucket. The CodeBuild should be constructed to source from the zip
file in the artifact bucket, no AWS CLI calls needed.

| Function | Description                                                                                                                                                                                                                                                                                                                                         |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| reset    | Functionality for Resetting an AWS Account using [aws-nuke](https://github.com/rebuy-de/aws-nuke). Updates the DynamoDB Account and Account Assignment manifest accordingly |

## Database Schema

**RedboxAccount<Namespace>** Table

Status of each Account in our pool

```
{
    "Id": "123456789012", # *Unique AWS Account ID*
    "GroupId": "5e27e7f0-1417-44b8-a19e-6a126b2db468" # *Unique Azure AD Group ID*
    "AccountStatus": "Assigned" | "Ready" | "NotReady"
    "LastModifiedOn": 1555690626 # *Epoch Timestamp*
}
```

Hash Key: `Id`
Range Key: `AccountStatus`

**RedboxAccountAssignment<Namespace>** Table

Current state of a users assignment to a AWS Account.
Records are unique by AccountId+UserId.

```
{
  "AccountId":  "123456789012", # AWS Account ID
  "UserId": "539133c1-db80-4606-9229-0ba49dfa0057" # *Azure AD Group ID*
  "AssignmentStatus": "Active" | "FinanceLock" | "ResetLock" | "ResetFinanceLock" | "Decommissioned"
  "CreatedOn": 1555690626 # *Epoch Timestamp*
  "LastModifiedOn": 1555690626 # *Epoch Timestamp*
}
```

Hash Key: `AccountId`
Range Key: `AssignmentStatus`

Secondary Index: UserId
Secondary Range Key: UserId

## Reset

AWS Redbox Reset will process an AWS Redbox Account to a clean and secure state.
The Reset has 2 main procedures, clearing the resources in an account (**Nuke**)
and reapply security monitoring (**Launchpad**).

The Reset of an account is done through a CodeBuild stage in a CodePipeline.

### Nuke

To clear resources from an AWS Redbox Account, [aws-nuke](https://github.com/rebuy-de/aws-nuke)
is used to list out all nuke-able resources and remove them. The configuration
file used to filter resources to not delete is located
[here](cmd/codepipeline/resetpipeline/redbox-nuke-all-config-template.yml).

### Alarms/Alerting

Cloudwatch alarms are defined in modules/alarms.tf, all alarms deliver to the SNS topic defined in modules/alarms_sns.tf.  
Alarms are defined based upon metrics available for each resource, [Metrics and Services](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/aws-services-cloudwatch-metrics.html).
They will vary by each service, please refer to the documentation above to create/modify Alarms/Alerts.

## Directory Structure

The following tree illustrates an example of the Directory Structure

```bash
aws_redbox
├── Jenkinsfile
├── Makefile
├── README.md
├── aws_account_info.txt
├── cmd
│   ├── codepipeline
│   │   └── resetpipeline
│   │       ├── buildspec.yml
│   │       ├── main.go
│   │       ├── main_test.go
│   │       └── redbox-nuke-all-config-template.yml
│   └── lambda
│       ├── acctmgr
│       │   ├── main.go
│       │   └── main_test.go
│       ├── financelock
│       │   ├── main.go
│       │   └── main_test.go
│       ├── resetsqs
│       │   ├── README.md
│       │   ├── main.go
│       │   ├── main_test.go
│       │   └── testdata
│       │       └── CloudWatch.json
│       └── resettrigger
│           └── main.go
├── docs
│   └── images
│       └── provision_decom_diagram.png
├── functions
│   └── adgroupadd
├── go.mod
├── go.sum
├── modules
│   ├── acctmgr.tf
│   ├── artifacts_bucket.tf
│   ├── backend.tf
│   ├── dynamodb.tf
│   ├── finance_lock.tf
│   ├── gateway.tf
│   ├── lambda_iam.tf
│   ├── main.tf
│   ├── outputs.tf
│   ├── reset.tf
│   ├── reset_codepipeline.tf
│   ├── swaggerRedbox.yaml
│   ├── variables.tf
│   ├── alarms_sns.tf
│   └── alarms.tf
├── pkg
│   ├── authorization
│   │   ├── authorizationer.go
│   │   └── mocks
│   │       └── Authorizationer.go
│   ├── common
│   │   ├── httpclient.go
│   │   ├── jwttoken.go
│   │   ├── mocks
│   │   │   ├── JWTTokenService.go
│   │   │   ├── Pipeline.go
│   │   │   └── Queue.go
│   │   ├── notification.go
│   │   ├── pipeline.go
│   │   ├── queue.go
│   │   ├── storage.go
│   │   ├── store.go
│   │   ├── token.go
│   │   └── util.go
│   ├── db
│   │   ├── db.go
│   │   ├── error.go
│   │   ├── mocks
│   │   │   └── DBer.go
│   │   └── model.go
│   ├── provision
│   │   ├── mocks
│   │   │   └── Provisioner.go
│   │   ├── provisioner.go
│   │   └── provisioner_test.go
│   ├── reset
│   │   ├── generateconfig.go
│   │   ├── generateconfig_test.go
│   │   ├── launchpad.go
│   │   ├── launchpad_test.go
│   │   ├── launchpader.go
│   │   ├── launchpader_test.go
│   │   ├── nuke.go
│   │   ├── nuke_test.go
│   │   ├── nuker.go
│   │   └── testdata
│   │       ├── test-config-result.yml
│   │       └── test-config-template.yml
│   ├── shell
│   │   └── shell.go
│   ├── terraform
│   │   ├── terraform.go
│   │   └── terraform_test.go
│   └── trigger
│       ├── reset.go
│       └── reset_test.go
├── scripts
│   ├── build.sh
│   └── deploy.sh
└── tests
    └── acceptance
        ├── artifacts_bucket_test.go
        ├── codepipeline_test.go
        ├── db_test.go
        └── outputs_test.go
```

## Account Provisioning & Decommissioning

![Account Provisioning & Decommissioning Diagram](/docs/images/provision_decom_diagram.png)

## API Spec

Redbox API Spec available via swaggerUI on github host: [API Spec](#)
