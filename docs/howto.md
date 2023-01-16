# How To

A practical guide to common operations and customizations for DCE.

## Use the DCE CLI

The DCE CLI is the easiest way to quickly deploy and use DCE. For more advanced usage, refer to the `DCE API <#use-the-dce-api>`_ section.

### Installing the DCE CLI

1. Download the appropriate executable for your OS from the [latest release](https://github.com/Optum/dce-cli/releases/latest). e.g. for mac, you should download dce_darwin_amd64.zip

1. Unzip the artifact and move the executable to a directory on your PATH, e.g.

    ```
    # Download the zip file
    wget https://github.com/Optum/dce-cli/releases/download/<VERSION>/dce_darwin_amd64.zip

    # Unzip to a directory on your path
    unzip dce_darwin_amd64.zip -d /usr/local/bin
    ```

1. Test the dce command by typing `dce`
    ```
    $ dce
    Disposable Cloud Environment (DCE) 

    The DCE cli allows:

    - Admins to provision DCE to a master account and administer said account
    - Users to lease accounts and execute commands against them

    Usage:
    dce [command]

    Available Commands:
    accounts    Manage dce accounts
    auth        Login to dce
    help        Help about any command
    init        First time DCE cli setup. Creates config file at "$HOME/.dce/config.yaml" (by default) or at the location specifief by "--config"
    leases      Manage dce leases
    system      Deploy and configure the DCE system
    usage       View lease budget information

    Flags:
        --config string   config file (default is "$HOME/.dce/config.yaml")
    -h, --help            help for dce

    Use "dce [command] --help" for more information about a command.
    ```

1. Type `dce init` to generate a new configuration file. Leave everything blank for now.

### Configuring AWS Credentials

The DCE CLI needs AWS IAM credentials any time it interacts with an AWS account. Below is a list of places where the DCE CLI
will look for credentials, ordered by precedence.

1. An API Token in the `api.token` field in the configuration file. You may obtain an API Token by:
    - Running the `dce auth` command
    - Base64 encoding the following JSON string. Note that `expireTime` is a Unix epoch timestamp and the string should
    not contain spaces or newline characters.

        ```json
        {
           "accessKeyId":"xxx",
           "secretAccessKey":"xxx",
           "sessionToken":"xxx",
           "expireTime":"xxx"
        }
        ```
1. The Environment Variables: `AWS_ACCESS_KEY_ID`, `AWS_ACCESS_KEY`, and `AWS_SESSION_TOKEN`
1. Stored in the AWS CLI credentials file under the `default` profile. This is located at `$HOME/.aws/credentials` on Linux/OSX and `%USERPROFILE%\.aws\credentials` on Windows.

.

### Deploying DCE from the CLI

You can build and deploy DCE from `source <#deploying-dce-from-source>`_ or by using the CLI.
This section will cover deployment using the DCE CLI with credentilas configured by the AWS CLI. See `Configuring AWS Credentials <#configuring-aws-credentials>`_ for alternatives.

1. [Download and install the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)

1. Choose an AWS account to be your new "DCE Master Account" and configure the AWS CLI with user credentials that have AdministratorAccess in that account.

    ```
    aws configure set aws_access_key_id default_access_key
    aws configure set aws_secret_access_key default_secret_key
    ```

1. Type `dce system deploy` to deploy dce to the AWS account specified in the previous step.

1. Edit your dce config file with the host and base url from the api gateway that was just deployed to your master account. This can be found in the master account under `API Gateway > (The API with "dce" in the title) > Stages > "Invoke URL: https://<host>/<baseurl>"`. Your config file should look something like this:

    ```
    api:
      host: abcdefghij.execute-api.us-east-1.amazonaws.com
      basepath: /api
    region: us-east-1
    ```

#### Using advanced deployment options

The DCE CLI uses [terraform](https://www.terraform.io/) to provision the infrastructure into the AWS account. 
You can use the `--tf-init-options` and `--tf-apply-options` to supply options directly to `terraform init`
and `terraform apply` (respectively) in the same format in which you would supply them to the `terraform` command.

> Note: if you are an advanced terraform user, you should consider 
> using the `DCE terraform module directly<terraform.html>`_.

The `--save-options` flag, if supplied, saves the values supplied to `--tf-init-options` and `--tf-apply-options`
in the configuration file in the following locations:

```yaml
terraform:
  initOptions: "-lock=true"
  applyOptions: "-compact-warnings -lock=true"
```

The DCE CLI stores its configuration by default in the `$HOME/.dce/config.yaml` location. This
can by overridden using the `--config` command line option. The file is as shown:

```yaml
# The API configutation. This is the DCE API that has been deployed to 
# an AWS account.
api:
  # This is the host name only, in the format of 
  # {restapi_id}.execute-api.{region}.amazonaws.com
  # For more information, see 
  # https://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-call-api.html
  host: api-gateway-id.execute-api.us-east-1.amazonaws.com
  # The stage name of the API Gateway
  # Default: /api
  basepath: /api
# The AWS region. It must match the region configured in the 
# api.host. Must be one of:
# "us-east-1", "us-east-2", "us-west-1", "us-west-2"
region: us-east-1
# Terraform configuration
terraform:
  # The full path to the locally-cached terraform binary used
  # by DCE to provision resources. Default value is 
  # $HOME/.dce/.cache/terraform/1.3.7/terraform
  bin: /path/to/terraform
  # The source from which terraform was downloaded. This
  # is reserved for future use.
  source: https://download.url.example.com/terraform.zip
  # The options passed to the underlying terraform init command.
  # This value is read if the --tf-init-options command option
  # is not specified or if the DCE_TF_INIT_OPTIONS environment
  # variable is empty, in that order.
  # The format of the value should be just as you would pass
  # them to terraform, as a quoted string.
  initOptions: ""
  # The options passed to the underlying terraform apply command.
  # Like the --tf-init-options flag, the command option is read
  # first, then the DCE_TF_APPLY_OPTIONS environment variable,
  # and lastly this value here. Use the --save-options flag to
  # easily save the values you supply on the CLI to this file.
  # The format of the value should be just as you would pass
  # them to terraform, as a quoted string.
  applyOptions: ""

```

### Authenticating with DCE

There are two ways to authenticate with DCE.

1. `Use custom IAM credentials for quick access to your individual DCE deployment <api-auth.html#using-iam-credentials>`_
1. `Use Cognito to set up admin and user profiles <./api-auth.html#using-aws-cognito>`_

### Adding a child account

1. Prepare a second AWS account to be your first "DCE Child Account".
    - Create an IAM role with `AdministratorAccess` and a trust relationship to your DCE Master Accounts
    - Create an account alias by clicking the 'customize' link in the IAM dashboard of the child account. This must not include the terms "prod" or "production".

1. Authenticate as an admin using the `dce auth` command if you are using `DCE with AWS Cognito <./api-auth.html#using-aws-cognito>`_

1. Use the `dce accounts add` command to add your child account to the "DCE Accounts Pool".

    **WARNING: This will delete any resources in the account.**

    ```
    dce accounts add --account-id 555555555555 --admin-role-arn arn:aws:iam::555555555555:role/DCEMasterAccess
    ```

1. Type `dce accounts list` to verify that your account has been added.

    ```
    dce accounts list
    [
        {
            "accountStatus": "NotReady",
            "adminRoleArn": "arn:aws:iam::555555555555:role/DCEMasterAccess",
            "createdOn": 1575485630,
            "id": "775788068104",
            "lastModifiedOn": 1575485630,
            "principalPolicyHash": "\"bc5872b50475b186afea67ff47516a8f\"",
            "principalRoleArn": "arn:aws:iam::775788768154:role/DCEPrincipal-quickstart"
        }
    ]
    ```
    The account status will initially say `NotReady`. It may take up to 5 minutes for the new account to be processed. Once the account status is `Ready`, you may proceed with creating a lease.

### Leasing a DCE Account

1. Now that your accounts pool isn't empty, you can create your first lease using the `dce leases create` command.

    ```
    dce leases create --budget-amount 100.0 --budget-currency USD --email jane.doe@email.com --principal-id quickstartuser
   Lease created: {
   	"accountId": "555555555555",
   	"budgetAmount": 100,
   	"budgetCurrency": "USD",
   	"budgetNotificationEmails": [
   		"jane.doe@email.com"
   	],
   	"createdOn": 1575509206,
   	"expiresOn": 1576114006,
   	"id": "19a742a0-149f-41e5-813a-6d3be101058b",
   	"lastModifiedOn": 1575509206,
   	"leaseStatus": "Active",
   	"leaseStatusModifiedOn": 1575509206,
   	"leaseStatusReason": "Active",
   	"principalId": "quickstartuser"
   }
   ```

1. Type `dce leases list` to verify that a lease has been created

    ```
    dce leases list
   [
   	{
   		"accountId": "555555555555",
   		"budgetAmount": 100,
   		"budgetCurrency": "USD",
   		"budgetNotificationEmails": [
   			"jane.doe@email.com"
   		],
   		"createdOn": 1575490207,
   		"expiresOn": 1576095007,
   		"id": "e501cb86-8317-458b-bdce-d47ab92f86a8",
   		"lastModifiedOn": 1575490207,
   		"leaseStatus": "Active",
   		"leaseStatusModifiedOn": 1575490207,
   		"leaseStatusReason": "Active",
   		"principalId": "quickstartuser"
   	}
   ]
    ```

### Logging into a leased account

There are three ways to "log in" to a leased account.

1. To use the *AWS CLI* with your leased account, type `dce leases login <lease-id>`. The `default` profile will be used unless you specify a different one with the `--profile` flag. 

    ```
    dce leases login --profile quickstart 19a742a0-149f-41e5-813a-6d3be101058b
    Adding credentials to .aws/credentials using AWS CLI
    cat ~/.aws/credentials
    [default]
    aws_access_key_id = xxxxxxxxxxxxxxxxxxxx
    aws_secret_access_key = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

    [quickstart]
    aws_access_key_id = xxxxxMAJKITANQZPFFXY
    aws_secret_access_key = xxxxxDEiaAvZ0OeqO5qxNBcJVrFGzNLxz6tgKWTF
    aws_session_token = xxxxxXIvYXdzEC0aDFEgMqpsBg4dtUS1qSKyAa3ktoh0SBPbwJv3S5B5NXdG8OdOVCQsya5b943mFfJnxX2reFw1a/r+LKa7G6CKj2NnWbkVWXdzWEVtsjy5Y32po2kVDp1lt74C7V6H8xbOk4HjgiXLOQl5faXpjmi80yaFI/yBrvnBbQVOq9QkbpeHcSyEkoouSkagCtkPicjLjq6omrAGR2xDXrrFYvYRIMevj2mZoBkk/5jGB3FpNycuWz6weqF4Z6qlCZLSalfetEAow7ml7wUyLf4OrtDvPgTPBjg6PClxC6BZgUMZaQM9ePQR0ZgMynNvm7JHbQz38jLCBqzneQ==
    ```

1. Access your leased account in a *web browser* via the `dce leases login` command with the `--open-browser` flag

    ```
    dce leases login --open-browser 19a742a0-149f-41e5-813a-6d3be101058b
    Opening AWS Console in Web Browser
    ```

1. To *print your credentials*, type `dce leases login` command with the `--print-creds` flag

    ```
    dce leases login --print-creds 19a742a0-149f-41e5-813a-6d3be101058b
    export AWS_ACCESS_KEY_ID=xxxxxMAJKITANQZPFFXY
    export AWS_SECRET_ACCESS_KEY=xxxxxDEiaAvZ0OeqO5qxNBcJVrFGzNLxz6tgKWTF
    export AWS_SESSION_TOKEN=xxxxxXIvYXdzEC0aDFEgMqpsBg4dtUS1qSKyAa3ktoh0SBPbwJv3S5B5NXdG8OdOVCQsya5b943mFfJnxX2reFw1a/r+LKa7G6CKj2NnWbkVWXdzWEVtsjy5Y32po2kVDp1lt74C7V6H8xbOk4HjgiXLOQl5faXpjmi80yaFI/yBrvnBbQVOq9QkbpeHcSyEkoouSkagCtkPicjLjq6omrAGR2xDXrrFYvYRIMevj2mZoBkk/5jGB3FpNycuWz6weqF4Z6qlCZLSalfetEAow7ml7wUyLf4OrtDvPgTPBjg6PClxC6BZgUMZaQM9ePQR0ZgMynNvm7JHbQz38jLCBqzneQ==
    ```

### Ending a Lease

1. End a lease using the `dce leases end` command with the `--account-id` and `--principal-id` flags

    ```
    dce leases end --account-id 555555555555 --principal-id jdoe99
    Lease ended
    ```

1. Type `dce leases list` to verify that the lease has been ended. The `leaseStatus` should now be marked as `Inactive`.

    ```
    dce leases list
   [
   	{
   		"accountId": "555555555555",
   		"budgetAmount": 100,
   		"budgetCurrency": "USD",
   		"budgetNotificationEmails": [
   			"jane.doe@email.com"
   		],
   		"createdOn": 1575490207,
   		"expiresOn": 1576095007,
   		"id": "e501cb86-8317-458b-bdce-d47ab92f86a8",
   		"lastModifiedOn": 1575490207,
   		"leaseStatus": "Inactive",
   		"leaseStatusModifiedOn": 1575490207,
   		"leaseStatusReason": "Destroyed",
   		"principalId": "quickstartuser"
   	}
   ]
    ```

### Removing a Child Account

1. Authenticate as an admin using the `dce auth` command if you are using `DCE with AWS Cognito <./api-auth.html#using-aws-cognito>`_

1. You can remove an account from the accounts pool using the `dce accounts remove` command

    ```
    dce accounts remove 555555555555
    ```

## Use the DCE API

DCE provides a set of endpoints for managing account pools and leases, and for monitoring account usage.

See [API Reference Documentation](./api-documentation.html) for details.

See [API Auth Documentation](./api-auth.html) for details on authenticating and authorizing requests.

### Prerequisites

Before you can deploy and use DCE, you will need the following:

1. An AWS account to use as the master account, and **sufficient credentials**
for deploying DCE into the account.
1. One or more AWS accounts to add as _child accounts_ in the account pool. 
DCE does not _create_ any AWS accounts for you. You will need to bring your own AWS accounts for adding to the account pool.
1. In each account you add to the account pool, you will create an IAM role
that allows DCE to control the child account.

### Deploying DCE from Source

You can build and deploy DCE from source or by using `the CLI <#deploying-dce-from-the-cli>`_.
This section will cover deployment from source. Please ensure you have the following:

1. [GNU Make](https://www.gnu.org/software/make/) 3.81+
1. [Go](https://golang.org/) 1.12.x+
1. Hashicorp [Terraform](https://www.terraform.io/) 1.3.7+
1. The [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) 1.16+

Once you have the requirements installed, you can deploy DCE into your account 
by following these steps:

1. Clone the [Github repository](https://github.com/Optum/dce) by using the 
command as shown here:

        $ git clone https://github.com/Optum/dce.git dce

1. Verify that the AWS CLI is [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html).
with an IAM user that has admin-level permissions in your AWS `master account <concepts.html#master-account>`_.
1. Make sure that the AWS region is set to *us-east-1* by using the command
as shown:

        $ aws configure list
            Name                    Value             Type    Location
            ----                    -----             ----    --------
        profile                <not set>             None    None
        access_key     ****************NXAW shared-credentials-file    
        secret_key     ****************ymwP shared-credentials-file    
        region                us-east-1      config-file    ~/.aws/config

1. Change into the base directory and use `make` to deploy the code as shown here:

        $ cd dce
        $ make deploy_local

When the last command is complete, you will have DCE deployed into your master
account.

### Finding the DCE API URL

The API is hosted by AWS API Gateway. The base URL is exposed as a Terraform output. API Gateway generates a unique ID as part of the API URL. To retrieve the base url of the API, run the following command from [the Terraform modules directory](https://github.com/Optum/dce/tree/master/modules):

```
terraform output api_url
```

All endpoints use this value as the base url. For example, to view accounts:

```
GET https://asdfghjkl.execute-api.us-east-1.amazonaws.com/api/accounts
```

### Authenticating with DCE

There are two ways to authenticate with DCE.

1. `Use custom IAM credentials for quick access to your individual DCE deployment <api-auth.html#using-iam-credentials>`_
1. `Use Cognito to set up admin and user profiles <./api-auth.html#using-aws-cognito>`_

### Adding Accounts to the DCE Account Pool

DCE manages its collection of AWS accounts in an `account pool <concepts.html#account-pool>`_. Each account in the pool is made available for `leasing <concepts.html#lease>`_ by DCE users.

DCE _does not_ create AWS accounts. These must be added to the account pool by a DCE administrator. You can create accounts using the AWS [CreateAccount API](https://docs.aws.amazon.com/cli/latest/reference/organizations/create-account.html).

The child account must have an administrative IAM Role with a trust relationship to allow the master account to assume the role. For example:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::MASTER_ACCOUNT_ID:root"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

Use the `/accounts` endpoint to add an account to the DCE accounts pool.

**Request**

`POST ${api_url}/accounts`
```json
{
    "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
    "id": "123456789012"
}
```

**Response**

```json
{
    "accountStatus": "NotReady",
    "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
    "createdOn": 1572379783,
    "id": "123456789012",
    "lastModifiedOn": 1572379783,
    "metadata": null,
    "principalPolicyHash": "\"852ee9abbf1220a111c435a8c0e65490\"",
    "principalRoleArn": "arn:aws:iam::123456789012:role/DCEPrincipal"
}
```

You can verify the account has been added with the following:

**Request**

`GET ${api_url}/accounts`

**Response**

```json
[
    {
        "accountStatus": "Ready",
        "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
        "createdOn": 1572379783,
        "id": "123456789012",
        "lastModifiedOn": 1572379888,
        "metadata": null,
        "principalPolicyHash": "\"852ee9abbf1220a111c435a8c0e65490\"",
        "principalRoleArn": "arn:aws:iam::123456789012:role/DCEPrincipal"
    }
]
```

### Leasing a child account

Now that the child account has been added to the account pool, you
can create a lease on the account.

**Request**

`POST ${api_url}/leases`
```json
{
    "principalId": "DCEPrincipal",
    "accountId": "123456789012",
    "budgetAmount": 20,
    "budgetCurrency": "USD",
    "budgetNotificationEmails": [
        "myuser@example.com"
    ],
    "expiresOn": 1572382800
}
```

**Response**

```json
{
    "accountId": "123456789012",
    "budgetAmount": 20,
    "budgetCurrency": "USD",
    "budgetNotificationEmails": [
        "myuser@example.com"
    ],
    "createdOn": 1572381585,
    "expiresOn": 1572382800,
    "id": "94503268-426b-4892-9b53-3c73ab38aeff",
    "lastModifiedOn": 1572381585,
    "leaseStatus": "Active",
    "leaseStatusModifiedOn": 1572381585,
    "leaseStatusReason": "",
    "principalId": "DCEPrincipal"
}
```

After getting the response, call the `/accounts` endpoint
again to see that the account status has been changed to
`Leased`:

**Request**

`GET ${api_url}/accounts`

**Response**

```json
[
    {
        "accountStatus": "Leased",
        "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
        "createdOn": 1572379783,
        "id": "123456789012",
        "lastModifiedOn": 1572381585,
        "metadata": null,
        "principalPolicyHash": "\"852ee9abbf1220a111c435a8c0e65490\"",
        "principalRoleArn": "arn:aws:iam::123456789012:role/DCEPrincipal"
    }
]
```

You may begin using your leased account once it's status has changed to `Leased`.

### Listing leases

You may list leases using the `/leases` endpoint

**Request**

`GET ${api_url}/leases`

**Response**

```json
[
    {
        "accountId": "123456789012",
        "budgetAmount": 20,
        "budgetCurrency": "USD",
        "budgetNotificationEmails": [
            "myuser@example.com"
        ],
        "createdOn": 1572381585,
        "expiresOn": 1572382800,
        "id": "94503268-426b-4892-9b53-3c73ab38aeff",
        "lastModifiedOn": 1572381585,
        "leaseStatus": "Active",
        "leaseStatusModifiedOn": 1572381585,
        "leaseStatusReason": "Active",
        "principalId": "DCEPrincipal"
    }
]
```

### Logging into a leased account

The easiest way to log into a leased account is by using the `DCE CLI <#logging-into-a-leased-account>`_. The following steps cover how to log in without using the CLI:

1. Configure `DCE Authentication <#authenticating-with-dce>`_ if you have not already done so
1. Open a web browser ([Google Chrome is recommended](https://github.com/Optum/dce/issues/166))
1. Navigate to `${api_url}/auth` and authenticate as prompted. You will be redirected to a page displaying an authentication code. 
1. Base64 decode the authentication code to view plaintext credentials of the form:

```json
{
   "accessKeyId":"xxx",
   "secretAccessKey":"xxx",
   "sessionToken":"xxx",
   "expireTime":"Wed Nov 20 2019 13:30:13 GMT-0600 (Central Standard Time)"
}
```

### Ending a lease

Leases automatically expire based on their expiration date or budget amount, but
leases may also be administratively destroyed at any time. To destroy
a lease with the API, send a DELETE request to the `/leases` endpoint.

**Request**

`DELETE ${api_url}/leases`
```json
{
    "principalId": "DCEPrincipal",
    "accountId": "123456789012"
}
```

**Response**

```json
{
    "accountId": "519777115644",
    "budgetAmount": 20,
    "budgetCurrency": "USD",
    "budgetNotificationEmails": [
        "john.doe@example.com"
    ],
    "createdOn": 1572381585,
    "expiresOn": 1572382800,
    "id": "94503268-426b-4892-9b53-3c73ab38aeff",
    "lastModifiedOn": 1572442028,
    "leaseStatus": "Inactive",
    "leaseStatusModifiedOn": 1572442028,
    "leaseStatusReason": "Destroyed",
    "principalId": "jdoe123"
}
```

## Configure Deployment Options

### Budgets and Lease Periods

Every `lease <concepts.html#lease>`_ comes with a configured **per-lease budget**, which limits AWS account spend during the course of the lease. Additionally there are **per-principal budgets**, which limit spend by a single user across multiple leases during a budget period. This prevents a single user from creating multiple leases as a way of circumventing lease budgets.

DCE budget may be configured as `Terraform variables <terraform.html#configuring-terraform-variables>`_.

| Variable | Default | Description |
| --- | --- | --- |
| `max_lease_budget_amount` | 1000 | The maximum budget a user may request for their lease |
| `max_lease_period` | 604800 | The maximum duration (seconds) a user may request for their lease |
| `principal_budget_amount` | 1000 | The maximum spend a user may accumulate across any number of leases during the `principal_budget_period` |
| `principal_budget_period` | "WEEKLY" | The period across which the `principal_budget_amount` is measured. Currently only supports "WEEKLY" |


### Account Resets

To `reset <concepts.html#reset>`_ AWS accounts between leases, DCE uses the [open source aws-nuke tool](https://github.com/rebuy-de/aws-nuke). This tool attempts to delete every single resource in th AWS account, and will make several attempts to ensure everything is wiped clean.

To prevent `aws-nuke` from deleting certain resources, provide a YAML configuration with a list of resource _filters_. (see [aws-nuke docs for the YAML filter configuration syntax](https://github.com/rebuy-de/aws-nuke#filtering-resources)). By default, DCE filters out resources which are critical to running DCE -- for example, the IAM roles for your account's `adminRoleArn` / `principalRoleArn`.

As a DCE implementor, you may have additional resources you wish protect from `aws-nuke`. If this is the case, you may specify your own custom `aws-nuke` YAML configuration:

- Copy the contents of [default-nuke-config-template.yml](https://github.com/Optum/dce/blob/master/cmd/codebuild/reset/default-nuke-config-template.yml) into a new file
- Modify as needed.
- Upload the YAML configuration file to an S3 bucket in the DCE master account

Then configure reset using `Terraform variables <terraform.html#configuring-terraform-variables>`_:

| Variable | Default | Description |
| --- | --- | --- |
| `reset_nuke_template_bucket` | See [default-nuke-config-template.yml](https://github.com/Optum/dce/blob/master/cmd/codebuild/reset/default-nuke-config-template.yml) | S3 bucket where a custom [aws-nuke](https://github.com/rebuy-de/aws-nuke) configuration is located |
| `reset_nuke_template_key` | See [default-nuke-config-template.yml](https://github.com/Optum/dce/blob/master/cmd/codebuild/reset/default-nuke-config-template.yml) | S3 key within the `reset_nuke_template_bucket` where a custom [aws-nuke](https://github.com/rebuy-de/aws-nuke) configuration is located |
| `reset_nuke_toggle` | `true` | Set to false to disable aws-nuke |
| `allowed_regions` | _all AWS regions_ | AWS regions which will be nuked. Allowing fewer regions will drastically reduce the run time of aws-nuke | 


### Budget Notifications

When a lease owner approaches or exceeds their budget, they will receive an email notification. These notifications are `configurable as Terraform variables <terraform.html#configuring-terraform-variables>`_:

| Variable | Default | Description |
| --- | --- | --- |
| `check_budget_enabled` | `true` | Set to `false` to disable budget checks entirely |
| `budget_notification_threshold_percentiles` | `[75, 100]` | Thresholds (percentiles) at which budget notification emails will be sent to users. |
| `budget_notification_from_email` | `"dce@example.com"` | `FROM` email address for budget notifications |
| `budget_notification_bcc_emails` | `[]` | Budget notifications emails will be BCC'd to these addresses |
| `budget_notification_template_subject` | See [variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) | Template for budget notification email subject |
| `budget_notification_template_text` | See [variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) | Template for budget notification text emails |
| `budget_notification_template_html` | See [variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) | Template for budget notification HTML emails |


#### Email Templates

Budget notification email templates are rendered using [golang templates](https://golang.org/pkg/text/template/), and accept the following arguments:

| Argument | Description |
| --- | --- |
| IsOverBudget | Set to `true` if the account is over the configured budget |
| Lease.PrincipalID | The principal ID of the lease holder |
| Lease.AccountID | The Account number of the AWS account in use |
| Lease.BudgetAmount | The configured budget amount for the lease |
| ActualSpend | The calculated spend on the account at time of notification |
| ThresholdPercentile | The configured threshold percentage for the notification |

### AWS Regions

By default, DCE users are limited to working in `us-east-1` by IAM Policy. Limiting users to a small number of regions reduces the amount of time it takes to reset accounts. 

To override this behavior, you may set the terraform `allowed_regions` variable to a list of AWS region names.

## Backup DCE Database Tables

DCE does not backup DynamoDB tables by default. However, if you want to restore a DynamoDB table from a backup, we do provide a helper script in [scripts/restore_db.sh](https://github.com/Optum/dce/blob/master/scripts/restore_db.sh). This script is also provided as a Github release artifact, for easy access.

To restore a DynamoDB table from a backup:

```
# Grab the account table name from Terraform state
table_name=$(cd modules && terraform output accounts_table_name)

# Or, grab the leases table name
table_name=$(cd modules && terraform output leases_table_name)

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

After restoring the DynamoDB table from a backup, `re-apply Terraform <terraform.html#deploy-with-terraform>`_ to ensure that your table is in sync with your Terraform configuration.

See [AWS guide for backing up DynamoDB tables](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Backup.Tutorial.html).

## Monitor DCE

### CloudWatch Dashboard

DCE comes with a prebuilt CloudWatch dashboard for monitoring things like API calls, account resets, and errors. To enable
the DCE CloudWatch Dashboard, set the `cloudwatch_dashboard_toggle` terraform variable to `true` during deployment. e.g.

```
terraform apply -var cloudwatch_dashboard_toggle=true
```

The DCE CloudWatch Dashboard is disabled by default.

### Account Pool Monitoring

DCE account pool monitoring may be enabled via the `account_pool_metrics_toggle` terraform variable. Account pool monitoring
publishes CloudWatch metrics on the number of accounts in each status (i.e. `Ready`, `Leased`, `NotReady`, and `Orphaned`).
The following CloudWatch alarms are included: 

* `ready-accounts`: triggers when the number of `Ready` accounts is below a configurable threshold. Controlled by the `ready_accounts_alarm_threshold` terraform variable.
* `orphaned-accounts`: triggers when the number of `Orphaned` accounts is above a configurable threshold. Controlled by the `orphaned_accounts_alarm_threshold` terraform variable.

To enable this feature with logical defaults, simply use:
```
terraform apply \
  -var account_pool_metrics_toggle=true \
  -var cloudwatch_dashboard_toggle=true \
```

DCE periodically queries the Accounts table to retrieve the number of accounts in each status. The frequency of these queries
can be controlled using the `account_pool_metrics_collection_rate_expression` terraform variable.

The DCE CloudWatch dashboard includes an `Account Pool` widget that displays the number of accounts in each status over time. 
The period over which metrics are aggregated in this widget can be controlled using the `account_pool_metrics_widget_period` terraform variable.

In order for data to display accurately in the `Account Pool` dashboard widget, the period of time over which data is aggregated 
in the widget (`account_pool_metrics_widget_period`) must be shorter than the metrics sampling interval (`account_pool_metrics_collection_rate_expression`).
Otherwise, multiple samples will be aggregated together in each data point.

For example, if `account_pool_metrics_collection_rate_expression` is set to `rate(30 minutes)`, then `1200` seconds (20 minutes)
 would be an acceptable value for `account_pool_metrics_widget_period`.

You may need to increase the DynamoDB Read Capacity Units on the Accounts table in order to accommodate this feature 
periodically querying all of the Account records. 13 RCUs per 100 accounts should be sufficient to avoid throttling. If needed,
 refer to the [AWS Documentation](https://aws.amazon.com/dynamodb/pricing/provisioned/) for assistance in 
calculating the required read capacity units appropriate for your usage.
This may be adjusted using the `accounts_table_rcu` terraform variable.

### CloudWatch Alarms

DCE also comes prebuilt with a number of CloudWatch alarms, which will trigger when DCE systems encounter errors or behave abnormally.

These CloudWatch alarms are delivered to an SNS topic. The ARN of this SNS topic is available as `a Terraform output <terraform.html#accessing-terraform-outputs>`_:

```
cd modules
terraform output alarm_sns_topic_arn
```

Subscribe to this topic to receive alarm notifications. For example:

```bash
aws sns subscribe \
  --topic-arn <Alarm Topic ARN> \
  --protocol email \
  --notification-endpoint my-email@example.com
``` 
