# AWS Redbox

# The premier repository for AWS Redbox.

<!--
Generated with markdown-toc
https://github.com/jonschlinkert/markdown-toc
markdown-toc -i README.md --maxdepth 3
-->

<!-- toc -->

- [What is Redbox?](#what-is-redbox)
- [Implementing Redbox](#implementing-redbox)
  * [Deploying Redbox](#deploying-redbox)
  * [Using the Redbox API](#using-the-redbox-api)
  * [Integrating with Redbox](#integrating-with-redbox)
  * [Integrating with Identity Providers](#integrating-with-identity-providers)
  * [Configuring aws-nuke](#configuring-aws-nuke)
- [Usage](#usage)
  * [Adding AWS Accounts to the Redbox Account Pool](#adding-aws-accounts-to-the-redbox-account-pool)
  * [Authenticating into Redbox Accounts](#authenticating-into-redbox-accounts)
- [API Reference](#api-reference)
  * [API Location](#api-location)
  * [Authorization](#authorization)
- [SNS Topic Reference](#sns-topic-reference)
  * [Account Created](#account-created)
  * [Account Deleted](#account-deleted)
  * [Lease Added](#lease-added)
  * [Lease Removed](#lease-removed)
  * [Lease Locked](#lease-locked)
  * [Lease Unlocked](#lease-unlocked)
- [Scripts](#scripts)
  * [`scripts/build.sh`](#scriptsbuildsh)
  * [`scripts/deploy.sh`](#scriptsdeploysh)
- [Build & Deploy](#build--deploy)
- [Database Schema](#database-schema)
- [Database Backups](#database-backups)
- [Reset](#reset)
  * [Nuke](#nuke)
  * [Alarms/Alerting](#alarmsalerting)
- [Account Provisioning & Decommissioning](#account-provisioning--decommissioning)
- [API Spec](#api-spec)
- [Notification via SES](#notification-via-ses)
- [Budget Features](#budget-features)
  * [Budget Notifications](#budget-notifications)

<!-- tocstop -->

## What is Redbox?

_TODO_

## Implementing Redbox

This repo provides a set of components which implementors (you) can use to deploy your own Redbox instance. These components come in the form of:

- Terraform modules to deploy Redbox infrastructure to AWS
- Packaged go modules and other assets, to deploy to Lambda, CodeBuild, etc. to your AWS master account

With these resources deployed, you will have access to a set of _integration points_, for working with your Redbox instance:

- APIs for managing Redbox resources (accounts, leases, etc.)
- SNS topics, allowing you hook in to Redbox events, and implement your own custom business logic

### Deploying Redbox

_TODO_

### Using the Redbox API

_TODO_

### Integrating with Redbox

Redbox provides a number of SNS topics, which allow you to hook into Redbox events, and implement your own custom business logic. Out of the box, Redbox is unopinionated about how you manage the details of your Redbox accounts. Some questions which are left to you to answer are:

- How do you grant and remove access to AWS Accounts?
- What do you do when an account reaches a budget threshold?

To answers to these questions, you can subscribe to SNS topics provided by Redbox. For example, you could subscribe to the _Lease Added_ topic, create an IAM User, and email an invite to the lease principal to login. On _Lease Removed_, you might delete that IAM User, and notify the lease principal that they no longer have access.

See the [SNS Topic Reference](#sns-topic-reference) for details on available SNS topics.

### Integrating with Identity Providers

When a new AWS account is added to the pool, [a role is created to allow principal users to login to the account](#adding-aws-accounts-to-the-redbox-account-pool), designated by the `adminRoleArn` field on the account object. By default, this role has an Assume Role Policy allowing IAM principals from the same account to assume it.

To integrate with alternative identity providers, you may [modify the Assume Role Policy on the IAM role.](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp.html). You may listen to events on the [account-created SNS topic](#account-created), which include the `principalRoleArn` in the message body.

#### Example: Vanilla Redbox Integration

_TODO: what's the simplest / least opinionated approach to integrating with Redbox_




### Configuring aws-nuke

In order to reset AWS accounts, Redbox uses the [open source `aws-nuke` tool](https://github.com/rebuy-de/aws-nuke). This tool tries its darndest to delete every single resource in your account, and will make several attempts to ensure everything is wiped clean.

To prevent `aws-nuke` from deleting certain resources, you may provide a YAML configuration with a list of resource _filters_. (see [`aws-nuke` docs for the YAML filter configuration syntax](https://github.com/rebuy-de/aws-nuke#filtering-resources)). By default, Redbox filters out resources which are critical to running Redbox -- for example, the IAM roles for your account's `adminRoleArn` / `principalRoleArn`.

As a Redbox implementor, you may have additional resources you wish protect from `aws-nuke`. If this is the case, you may specify your own custom `aws-nuke` YAML configuration:

- Copy the contents of [`default-nuke-config-template.yml`](./cmd/codebuild/reset/default-nuke-config-template.yml) into your own file, and modify as needed.
  - See [`aws-nuke` docs for the YAML `filters` configuration syntax](https://github.com/rebuy-de/aws-nuke#filtering-resources) 
- Upload your YAML configuration file to an S3 bucket
- Set the Terraform `reset_nuke_template_bucket` and `reset_nuke_template_key` to point at your YAML configuration file on S3
- Make sure [you have aws-nuke enabled](#enabling-aws-nuke)

#### Template parameters for aws-nuke YAML configuration

Redbox allows you to use a number of template parameters within your `aws-nuke` YAML config, which will be resolved a runtime:

| Parameter | Description |
| ------ | ------ |
| `{{id}}` |  The AWS Account ID, for the account currently being nuked |
| `{{admin_role}}` |  The name of the IAM role assumed by the Redbox master account to manage the child account. |
| `{{principal_role}}` |  The name of the IAM role assumed by end users of Redbox, in order to login to their AWS account |
| `{{principal_policy}}` | The IAM policy assigned the the `principal_role` |



#### Enabling aws-nuke

By default, `aws-nuke` is set to execute in _Dry Run_ mode, so that you don't accidentally destroy critical resources in your AWS account. To enable `aws-nuke`, you will need to set the Terraform `reset_nuke_toggle` variable to `"true"`. 

## Usage

### Adding AWS Accounts to the Redbox Account Pool

To add an account to the Redbox Account Pool, you may use the `POST /accounts` endpoint.

eg.

```
POST /accounts
{
  "id": "123456789012"
  "adminRoleArn": "arn:aws:iam::123456789012:role/RedboxAdmin"
}

Response:
{
  "id": "1234567890",
  "accountStatus": "NotReady",
  "adminRoleArn": "arn:aws:iam::1234567890123:role/adminRole",
  "principalRoleArn":  "arn:aws:iam::1234567890123:role/RedboxPrincipal",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "metadata": {}
}
```

The IAM Role passed in as `adminRoleArn` must be assumable by the Redbox master account, and have appropriate IAM access to manage the Redbox Account (eg. can run [aws-nuke](https://github.com/rebuy-de/aws-nuke) in the account).

When adding the account to the pool, the account will be marked as `NotReady`, and queued for reset. You will need to wait for reset to complete and the account to be marked as `Ready` before requesting leases against the account.

Redbox will create a new IAM Role to be assumed by principal users of the account. The ARN for this role will be included in the response, as `principalRoleArn`. The principal's role has near-admin access to the account, with the following exceptions:

- Cannot create resources which cannot be deleted by Redbox
- Cannot create support tickets, or increase service limits
- Is restricted to `us-east-1` and `us-west-1`
- Cannot modify resources managed by Redbox (including itself)

See [_Integrating with Identity Providers_](#integrating-with-identity-providers) for documentation on assuming the Redbox principal role using an identity provider.

### Authenticating into Redbox Accounts

## API Reference

Redbox exposes an API for managing Redbox accounts and leases.

See [swaggerRedbox.yaml](./modules/swaggerRedbox.yaml) for endpoint documentation (better Swagger docs to come...).

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

## SNS Topic Reference

### Account Created

#### Description

An account was added to the account pool

#### Payload

This message includes a payload as JSON, with the following fields:

| Field          | Type                             | Description                                                                                                 |
| -------------- | -------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| id             | string                           | AWS Account ID                                                                                              |
| accountStatus  | "Ready", "NotReady", or "Leased" | Account status                                                                                              |
| adminRoleArn   | string                           | ARN for the IAM role used by the Redbox master account to manage the account                                |
| lastModifiedOn | int                              | Last modified timestamp                                                                                     |
| createdOn      | int                              | Last modified timestamp                                                                                     |
| metadata       | JSON object                      | Metadata field contains any organization specific data pertaining to the account that needs to be persisted |


Example:

```json
{
  "id": "1234567890",
  "accountStatus": "NotReady",
  "adminRoleArn": "arn:aws:iam::1234567890123:role/adminRole",
  "principalRoleArn": "arn:aws:iam::1234567890123:role/RedboxPrincipal",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "metadata": {}
}
```

#### Topic ARN

This SNS topic ARN is provided as a Terraform output:

```
terraform output account_created_topic_arn
```

### Account Deleted

#### Description

An account was deleted from the account pool

#### Topic ARN

This SNS topic ARN is provided as a Terraform output:

```
terraform output account_deleted_topic_arn
```

### Lease Added

#### Description

Triggered when a lease is created.

#### Payload

This message includes a payload as JSON, with the following fields:

| Field           | Type    | Description                                         |
| --------------- | ------- | --------------------------------------------------- |
| accountId       | string  | AWS Account ID                                      |
| principalId     | string  | ID of the principal user, associated with the lease |
| leaseStatus     | string  | Status of the lease.                                |
| createdOn       | integer | Timestamp (epoch) of creation                       |
| lastModifiedOn  | integer | Timestamp (epoch) of last modification              |
| leaseModifiedOn | integer | Timestamp (epoch) of lease status modification      |

Example:

```json
{
  "accountId": "1234567890",
  "principalId": "jdoe17",
  "leaseStatus": "Active",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "leaseStatusModifiedOn": 1560306008
}
```

#### Topic ARN

This SNS topic ARN is provided as a Terraform output:

```
terraform output lease_added_topic_arn
```

### Lease Removed

#### Description

Triggered when a lease is deleted.

#### Payload

This message includes a payload as JSON, with the following fields:

| Field                 | Type    | Description                                         |
| --------------------- | ------- | --------------------------------------------------- |
| accountId             | string  | AWS Account ID                                      |
| principalId           | string  | ID of the principal user associated with the lease  |
| leaseStatus           | string  | Status of the lease.                                |
| createdOn             | integer | Timestamp (epoch) of creation                       |
| lastModifiedOn        | integer | Timestamp (epoch) of last modification              |
| leaseStatusModifiedOn | integer | Timestamp (epoch) of last lease status modification |

Example:

```json
{
  "accountId": "1234567890",
  "principalId": "jdoe17",
  "leaseStatus": "Decommissioned",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "leaseStatusModifiedOn": 1560306008
}
```

#### Topic ARN

This SNS topic ARN is provided as a Terraform output:

```
terraform output lease_removed_topic_arn
```

### Lease Locked

#### Description

Triggered when a lease is "locked". Locking a lease means that the principal's access to the account has been temporarily disabled. For example, a lease will be locked when the AWS account reaches it's max budget threshold, and unlocked again after the end of the lease period.

AWS Redbox is unopinionated about how lease locks are implemented. It is up to you on how you want to respond to this topic (eg. by removing the principal's access to the account). 

#### Payload

This message payload is the lease object as JSON, with the following fields:

| Field           | Type    | Description                                         |
| --------------- | ------- | --------------------------------------------------- |
| accountId       | string  | AWS Account ID                                      |
| principalId     | string  | ID of the principal user, associated with the lease |
| leaseStatus     | string  | Status of the lease.                                |
| createdOn       | integer | Timestamp (epoch) of creation                       |
| lastModifiedOn  | integer | Timestamp (epoch) of last modification              |
| leaseModifiedOn | integer | Timestamp (epoch) of lease status modification      |

Example:

```json
{
  "accountId": "1234567890",
  "principalId": "jdoe17",
  "leaseStatus": "Active",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "leaseStatusModifiedOn": 1560306008
}
```

#### Topic ARN

This SNS topic ARN is provided as a Terraform output:

```
terraform output lease_locked_topic_arn
```

### Lease Unlocked

#### Description

Triggered when a lease is "unlocked". Locking a lease means that the principal's access to the account has been temporarily disabled. For example, a lease will be locked when the AWS account reaches it's max budget threshold, and unlocked again after the end of the lease period.

AWS Redbox is unopinionated about how lease locks are implemented. It is up to you on how you want to respond to this topic (eg. by removing the principal's access to the account). 

#### Payload

This message payload is the lease object as JSON, with the following fields:

| Field           | Type    | Description                                         |
| --------------- | ------- | --------------------------------------------------- |
| accountId       | string  | AWS Account ID                                      |
| principalId     | string  | ID of the principal user, associated with the lease |
| leaseStatus     | string  | Status of the lease.                                |
| createdOn       | integer | Timestamp (epoch) of creation                       |
| lastModifiedOn  | integer | Timestamp (epoch) of last modification              |
| leaseModifiedOn | integer | Timestamp (epoch) of lease status modification      |

Example:

```json
{
  "accountId": "1234567890",
  "principalId": "jdoe17",
  "leaseStatus": "Active",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "leaseStatusModifiedOn": 1560306008
}
```

#### Topic ARN

This SNS topic ARN is provided as a Terraform output:

```
terraform output lease_unlocked_topic_arn
```

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

## Database Schema

**RedboxAccount<Namespace>** Table

Status of each Account in our pool

```
{
    "Id": "123456789012", # *Unique AWS Account ID*
    "AccountStatus": "Leased" | "Ready" | "NotReady"
    "LastModifiedOn": 1555690626 # *Epoch Timestamp*
}
```

Hash Key: `Id`
Range Key: `AccountStatus`

**RedboxLease<Namespace>** Table

Current state of a users lease to a AWS Account.
Records are unique by AccountId+PrincipalId.

```
{
  "AccountId":  "123456789012", # AWS Account ID
  "PrincipalId": "098765432"
  "LeaseStatus": "Active" | "FinanceLock" | "ResetLock" | "ResetFinanceLock" | "Decommissioned"
  "CreatedOn": 1555690626 # *Epoch Timestamp*
  "LastModifiedOn": 1555690626 # *Epoch Timestamp*
  "BudgetAmount": 300
  "BudgetCurrency": "USD"
  "BUdgetNotificationEmail": ["user@test.com", "manager@test.com"]
}
```

Hash Key: `AccountId`
Range Key: `LeaseStatus`

Secondary Index: PrincipalId
Secondary Range Key: PrincipalId

## Database Backups

Redbox does not backup your DynamoDB tables by default. However, if you want to restore a DynamoDB table from a backup, we do provide a helper script in [scripts/restore_db.sh](./scripts/restore_db.sh). This script is also provided as a Github release artifact, for easy access.

To restore a DynamoDB table from a backup:

```
# Grab the account table name from Terraform state
table_name=$(cd modules && terraform output redbox_account_db_table_name)

# Or, grab the leases table name
table_name=$(cd modules && terraform output redbox_lease_db_table_name)

# List available backups
./scripts/restore_db.sh \
  --target-table-name ${table_name} \
  --list-backups

# Choose an backup from the output of the last command, and pass in the ARN
./scripts/restore_db.sh \
  --target-table-name ${table_name} \
  --backup-arn <backup arn>

# If the table already exists, and you want to delete and
# recreate it from a backup, pass in
# the --force-delete-table flag
./scripts/restore_db.sh \
  --target-table-name ${table_name} \
  --backup-arn <backup arn> \
  --force-delete-table
```

After restoring your DynamoDB table from a backup, you should rerun `terraform apply` to ensure that your table is in sync with your Terraform configuration.

## Reset

AWS Redbox Reset will process an AWS Redbox Account to a clean and secure state.
The Reset has 2 main procedures, clearing the resources in an account (**Nuke**)
and reapply security monitoring (**Launchpad**).

The Reset of an account is done through a CodeBuild stage in a CodePipeline.

### Nuke

To clear resources from an AWS Redbox Account, [aws-nuke](https://github.com/rebuy-de/aws-nuke)
is used to list out all nuke-able resources and remove them. The defualt
configuration file used to filter resources to not delete is located
[here](cmd/codebuild/reset/default-nuke-config-template.yml). The
configuration file can also be pulled from an S3 Bucket Object via setting
the `RESET_NUKE_TEMPLATE_BUCKET` and `RESET_NUKE_TEMPLATE_KEY`, these are
default to `STUB` and are ignored.

### Alarms/Alerting

Cloudwatch alarms are defined in modules/alarms.tf, all alarms deliver to the SNS topic defined in modules/alarms_sns.tf.  
Alarms are defined based upon metrics available for each resource, [Metrics and Services](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/aws-services-cloudwatch-metrics.html).
They will vary by each service, please refer to the documentation above to create/modify Alarms/Alerts.

## Account Provisioning & Decommissioning

![Account Provisioning & Decommissioning Diagram](/docs/images/provision_decom_diagram.png)

## API Spec

Redbox API Spec available via swaggerUI on github host: [API Spec](https://github.optum.com/pages/CommercialCloud-Team/aws_redbox/)

## Notification via SES

This repo makes use of the Simple Email Service from AWS, which requires a verified email addrss for the functionality being used.
To allow for this, the email address configured in the terraform requires a confirmation to made manually upon the applicaiton of the email
address to the account.

The address MUST reply to a confirmation email sent from SES to verify the account before emails can commence.  

## Budget Features

### Budget Notifications

Budget notifications will be sent out of a verified Simple Email Service account.  Verification of this address is a manual process, see above
"Notification via SES" section.
Some variables used in notification templates (conatined in modules/variables.tf):
  - IsOverBudget : Boolean determining account budget status
  - Lease.PrincipalID : The UserID of the lease holder
  - Lease.AccountID : The Account number of the AWS account in use
  - Lease.BudgetAmount : The configured budget amount for the lease
  - ActualSpend : The calculated spend on the account at time of notification
  - ThresholdPercentile : The conigured threshold percentage for the notification, prior to exhaustion