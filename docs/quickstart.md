# DCE Quickstart

The purpose of this quickstart is to show how to deploy DCE into
a _master account_ and to show you how to add other AWS accounts
into the _account pool_. To understand more about _master accounts_,
_child accounts_, _account pools_, and _leases_, see [the concepts documentation](concepts.md).

## Prerequisites

Before you can deploy and use DCE, you will need the following:

1. An AWS account to use as the master account, and **sufficient credentials**
for deploying DCE into the account.
1. One or more AWS accounts to add as _child accounts_ in the account pool. As 
of the time of this writing, DCE does not _create_ any AWS accounts for you. 
You will need to bring your own AWS accounts for adding to the account pool.
1. In each account you add to the account pool, you will create an IAM role
that allows DCE to control the child account. This is detailed later 
in this quickstart.

## Basic steps (using the REST API)

To deploy and start using DCE, follow these basic steps (explained in 
greater detail below):

1. Deploy DCE to your master account.
1. Provision the IAM role in the child account.
1. Add each account to the account pool by using the 
[CLI](https://github.com/Optum/dce-cli) or [REST API](api-documentation.md).

Each of these steps is covered in greater detail in the sections below.

## Using the CLI

To deploy DCE into the "master" account, you will need to start out with
an existing AWS account. The `dce` [command line interface](https://github.com/Optum/dce-cli) (CLI) 
is the easiest way to deploy DCE into your master account.

## Using the REST API

### Deploying DCE to the master account

You can also download DCE from the [Github repository](https://github.com/Optum/dce)
and install it directly. To do so, you will need the following installed:

1. [GNU Make](https://www.gnu.org/software/make/) 3.81+
1. [Go](https://golang.org/) 1.12.x+
1. Hashicorp [Terraform](https://www.terraform.io/) 0.12+
1. The [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) 1.16+

Once you have the requirements installed, you can deploy DCE into your account 
by following these steps:

1. Clone the [Github repository](https://github.com/Optum/dce) by using the 
command as shown here:

        $ git clone https://github.com/Optum/dce.git dce

1. Verify that the AWS CLI is [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html).
with an IAM user that has admin-level permissions in your AWS 
[master account](concepts.md#master-account).
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

### Adding child accounts to the account pool

To create an account using the API, use an HTTP POST to the URL 
*${api_url}/accounts* with the following content:

```json
{
    "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
    "id": "123456789012"
}
```

The response will be as shown:

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

And you can verify the account is there by using the API, this time
with an HTTP GET *${api_url}/accounts* to get the following response:

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

### Leasing the child account

Now that the child account has been added to the account pool, you
can create a lease on the account. HTTP POST the following 
content to the *${api_url}/leases* endpoint:

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

You will see a response that looks like this:

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

After getting the response, call the *${api_url}/accounts* endpoint 
again to see that the account status has been changed to 
`Leased`:

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
Once you see the first lease provisioned in the system, you are ready to use your
first lease! See [logging into your leased account](howto.md#login-to-your-dce-account).

### Listing leases

To list the leases, use an HTTP GET request to the *${api_url}/leases* endpoint
to see the response as shown here:

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

### Destroying leases

Leases can automatically expire based on a date or a budget amount, but
leases may also be administratively destroyed at any time. To destroy
a lease with the API, send a DELETE request to *${api_url}/leases
with the following request body:

```json
{
    "principalId": "DCEPrincipal",
    "accountId": "123456789012"
}
```

The API response for a successful lease destroy looks like this:

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
