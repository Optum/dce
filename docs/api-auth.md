# DCE API Authentication / Authorization


There are two ways to authenticate against the DCE APIs:

1. [Using Cognito](#using-cognito)
1. [Using IAM credentials](#using-iam-credentials)

## Using Cognito

Cognito is used to allow an admin to add users to DCE.  This can be done by using the Cognito User Pool.  Additionally any IdP supported by Cognito User Pools can also be used.

### Roles

#### Admins

Admins have full control to all APIs and will not get back filtered results when querying APIs. 

There are three different ways a user is considered an admin:

1. They have an IAM user/role/etc with a policy that gives them access to the API
1. A Cognito user is placed into a Cognito group called `Admins`
1. A Cognito user has an attribute in `custom:roles` that will match a search criteria specified by the Terraform variable `cognito_roles_attribute_admin_name`

#### Users

Users (by default) are given access to the leasing and usage APIs.  This is done so they can request their own lease and look at the usage of their leases.  Any appropriately authenticated user in Cognito will automatically fall into the `Users` role.

## Using IAM Credentials


The DCE API accepts authentication via IAM credentials using [SigV4 signed requests](https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html).

Any requests made via IAM Credentials will be treated as an [admin role](#admins). 

The process for signing requests with SigV4 is somewhat involved, but luckily there are a number of tools to make this easier. For example:

- [AWS Golang SDK `signer/v4` package](https://docs.aws.amazon.com/sdk-for-go/api/aws/signer/v4/)
- [aws-requests-auth](https://github.com/DavidMuller/aws-requests-auth) for Python
- [Postman _AWS Signature_ authentication](https://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-use-postman-to-call-api.html)

AWS also provides [examples for a number of languages in their docs](https://docs.aws.amazon.com/general/latest/gr/signature-v4-examples.html).



### IAM Policy for DCE API requests

The IAM principal used to send requests to the DCE API must have sufficient permissions to execute API requests.

The Terraform module in the repo provides an IAM policy with appropriate permissions for executing DCE API requests. The policy name and ARN are available as Terraform outputs.

```
terraform output api_access_policy_name
terraform output api_access_policy_arn
```