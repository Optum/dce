# DCE Quickstart

The purpose of this quickstart is to show how to deploy DCE into
a _master account_ and to show you how to add other AWS accounts
into the _account pool_. To understand more about _master accounts_,
_child accounts_, _account pools_, and _leases_, see [the glossary](/glossary/).

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
[CLI](/using-the-cli/) or [REST API](/api-documentation/).

Each of these steps is covered in greater detail in the sections below.

## Using the CLI

To deploy DCE into the "master" account, you will need to start out with
an existing AWS account. The `dce` [command line interface](dce/dce/) (CLI) 
is the easiest way to deploy DCE into your master account.

## Using the REST API

### Deploying DCE to the master account

You can also download DCE from the [Github repository](https://github.com/Optum/Redbox)
and install it directly. To do so, you will need the following installed:

1. [GNU Make](https://www.gnu.org/software/make/) 3.81+
1. [Go](https://golang.org/) 1.12.x+
1. Hashicorp [Terraform](https://www.terraform.io/) 0.12+
1. The [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) 1.16+

Once you have the requirements installed, you can deploy DCE into your account 
by following these steps:

1. Clone the [Github repository](https://github.com/Optum/Redbox) by using the 
command as shown here:

        $ git clone https://github.com/Optum/Redbox.git dce

1. Verify that the AWS CLI is [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html).
with an IAM user that has admin-level permissions in your AWS 
[master account](/glossary/#master-account).
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
HTTP/1.1 201 Created
Access-Control-Allow-Origin: *
Connection: keep-alive
Content-Length: 312
Content-Type: application/json
Date: Tue, 29 Oct 2019 20:09:44 GMT
Via: 1.1 5175c0b4bfaddbfcca703b6ef1dc6bad.cloudfront.net (CloudFront)
X-Amz-Cf-Id: DYwJXcmH3UJ3yLFdB7CYfxI-6szcpYKF-soPz9pesI6JmU-Nu3t2xg==
X-Amz-Cf-Pop: ORD52-C1
X-Amzn-Trace-Id: Root=1-5db89c87-9bef5e560d1c2129624430d3;Sampled=0
X-Cache: Miss from cloudfront
x-amz-apigw-id: CV1lOEOwIAMFQag=
x-amzn-RequestId: 5ef9f032-bed2-45e2-abf8-7051d15ef966

{
    "accountStatus": "NotReady",
    "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
    "createdOn": 1572379783,
    "id": "123456789012",
    "lastModifiedOn": 1572379783,
    "metadata": null,
    "principalPolicyHash": "\"852ee9abbf1220a111c435a8c0e65490\"",
    "principalRoleArn": "arn:aws:iam::123456789012:role/RedboxPrincipal-myuser"
}
```

And you can verify the account is there by using the API, this time
with an HTTP GET *${api_url}/accounts* to get the following response:

```json
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Connection: keep-alive
Content-Length: 311
Content-Type: application/json
Date: Tue, 29 Oct 2019 20:15:36 GMT
Via: 1.1 613fc2ce2843d97a87bffbdb759c82a5.cloudfront.net (CloudFront)
X-Amz-Cf-Id: hZcEUzHmgnCCXX6rUI5Lc-W3CTIqkb7Sjh01lVZtti7b41mGbvpuSg==
X-Amz-Cf-Pop: ORD52-C1
X-Amzn-Trace-Id: Root=1-5db89de8-667c3c90d45d54d08e96e9d0;Sampled=0
X-Cache: Miss from cloudfront
x-amz-apigw-id: CV2cVGJSIAMFb8w=
x-amzn-RequestId: ab46c728-60cf-48ee-a838-e0d076025667

[
    {
        "accountStatus": "Ready",
        "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
        "createdOn": 1572379783,
        "id": "123456789012",
        "lastModifiedOn": 1572379888,
        "metadata": null,
        "principalPolicyHash": "\"852ee9abbf1220a111c435a8c0e65490\"",
        "principalRoleArn": "arn:aws:iam::123456789012:role/RedboxPrincipal-myuser"
    }
]
```

### Leasing the child account

Now that the child account has been added to the account pool, you
can create a lease on the account. HTTP POST the following 
content to the *${api_url}/leases* endpoint:

```json
{
    "principalId": "RedboxPrincipal-myuser",
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
HTTP/1.1 201 Created
Connection: keep-alive
Content-Length: 372
Content-Type: application/json
Date: Tue, 29 Oct 2019 20:39:46 GMT
Via: 1.1 05ce646a2ff6febe063c256476b18a9c.cloudfront.net (CloudFront)
X-Amz-Cf-Id: Yo5Q0vd4yReiQI_tD4IVvpkP7HgFqUk3X3DJ8wLeR0Gpl0mh_jYkxg==
X-Amz-Cf-Pop: ORD52-C2
X-Amzn-Trace-Id: Root=1-5db8a391-5577d7a477be61bde1c39308;Sampled=0
X-Cache: Miss from cloudfront
x-amz-apigw-id: CV5-yF_coAMFd4A=
x-amzn-RequestId: f4848ef9-d577-4465-a0d2-dd33e792f4a5

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
    "principalId": "RedboxPrincipal-myuser"
}
```

After getting the response, call the *${api_url}/accounts* endpoint 
again to see that the account status has been changed to 
`Leased`:

```json
HTTP/1.1 200 OK
// snipped...
[
    {
        "accountStatus": "Leased",
        "adminRoleArn": "arn:aws:iam::123456789012:role/DCEAdmin",
        "createdOn": 1572379783,
        "id": "123456789012",
        "lastModifiedOn": 1572381585,
        "metadata": null,
        "principalPolicyHash": "\"852ee9abbf1220a111c435a8c0e65490\"",
        "principalRoleArn": "arn:aws:iam::123456789012:role/RedboxPrincipal-myuser"
    }
]
```
Once you see the first lease provisioned in the system, you are ready to use your
first lease! See [logging into your leased account](#logging-into-your-leased-account).

### Listing leases

To list the leases, use an HTTP GET request to the *${api_url}/leases* endpoint
to see the response as shown here:

```json
HTTP/1.1 200 OK
Connection: keep-alive
Content-Length: 380
Content-Type: application/json
Date: Wed, 30 Oct 2019 12:47:46 GMT
Via: 1.1 0e31b6655e8230805e58fd71c1351ba1.cloudfront.net (CloudFront)
X-Amz-Cf-Id: pGPruQ4PEi2KGd3O-R2eUl48fuEUesNdR1UhOnrNgf_fYlJ0SQwzaA==
X-Amz-Cf-Pop: ORD52-C1
X-Amzn-Trace-Id: Root=1-5db98671-b6dd22900fc6c8367b3b6084;Sampled=0
X-Cache: Miss from cloudfront
x-amz-apigw-id: CYHxsH2VoAMFwxQ=
x-amzn-RequestId: d306e800-dfd1-4a53-88b1-2a5d469c9cd6

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
        "principalId": "RedboxPrincipal-myuser"
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
    "principalId": "RedboxPrincipal-myuser",
    "accountId": "123456789012"
}
```

The API response for a successful lease destroy looks like this:

```json
HTTP/1.1 200 OK
// snipped...
{
    "accountId": "519777115644",
    "budgetAmount": 20,
    "budgetCurrency": "USD",
    "budgetNotificationEmails": [
        "nathan@galenhousesoftware.com"
    ],
    "createdOn": 1572381585,
    "expiresOn": 1572382800,
    "id": "94503268-426b-4892-9b53-3c73ab38aeff",
    "lastModifiedOn": 1572442028,
    "leaseStatus": "Inactive",
    "leaseStatusModifiedOn": 1572442028,
    "leaseStatusReason": "Destroyed",
    "principalId": "RedboxPrincipal-nathangood"
}
```
