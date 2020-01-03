# API Auth

There are two ways to authenticate against the DCE APIs:

1. [Using Cognito](#using-cognito)
1. [Using IAM credentials](#using-iam-credentials)

## Using AWS Cognito

AWS Cognito is used to authenticate and authorize DCE users. This section will walk through setting this
up in the AWS web console, but note that all of these operations can be automated using the AWS CLI or SDKs. While
this example uses Cognito User Pools to create and manage users, you may also [integrate Cognito with your own IdP](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pools-identity-provider.html).

### Roles

#### Admins

Admins have full access to all APIs and will not get back filtered results when querying APIs.

There are three different ways a user is considered an admin:

1. They have an IAM user/role/etc with a policy that gives them access to the API
1. A Cognito user is placed into a Cognito group called `Admins`
1. A Cognito user has an attribute in `custom:roles` that will match a search criteria specified by the Terraform variable `cognito_roles_attribute_admin_name`

#### Users

Users (by default) are given access to the leasing and usage APIs.  This is done so they can request their own lease and look at the usage of their leases.  Any appropriately authenticated user in Cognito will automatically fall into the `Users` role.

### Configuring Cognito

1. Open the AWS Console in your DCE Master Account and Navigate to AWS Cognito by typing `Cognito` in the search bar

    ![Cognito](./img/cognito.png)

1. Select `Manage User Pools` and click on the dce user pool.

    ![Manage user pools](./img/manageuserpools.png)

1. Select `Users and groups`

    ![Users and Groups](./img/usersandgroups.png)

1. Create a user

    ![Create User](./img/createuser.png)

1. Name the user and provide a temporary password. You may uncheck all of the boxes and leave the other fields blank. This user will not have admin priviliges.

    ![Quick start user](./img/quickstartuser.png)

1. Create a second user to serve as a system admin. Follow the same steps as you did for creating the first user, but name this one something appropriate for their role as an administrator.

    ![Quick start admin](./img/quickstartadmin.png)

1. Create a group

    ![Create group](./img/creategroup.png)

1. Users in this group will be granted admin access to DCE. The group name must contain the term `Admin`. Choose a name and click on the `Create group` button.

    ![Group name](./img/groupname.png)

1. Add your admin user to the admin group to grant them admin privileges.
    ![Quick start admin detail](./img/quickstartadmindetail.png)
    ![Add to group](./img/addtogroup.png)

1. Type `dce auth` in your command terminal. This will open a browser with a log in screen. Enter the username and password for the non-admin user that you created. Reset the password as prompted.

    ![Quick start user login](./img/quickstartuserlogin.png)

1. Upon successfully logging in, you will be redirected to a credentials page containing a temporary authentication code. Click the button to copy the auth code to your clipboard.

    ![Credentials Page](./img/credspage.png)

1. Return to your command terminal and paste the auth code into the prompt.

    ```
    dce auth
    ✔ Enter API Token: : █
    ```

1. You are now authenticated as a DCE User. Test that you have proper authorization by typing `dce leases list`.
This will return an empty list indicating that there are currently no leases which you can view.
If you are not properly authenticated as a user, you will see a permissions error.

    ```
    dce leases list
    []
    ```

1. Users are not authorized to list child accounts in the accounts pool. Type `dce accounts list` to verify that you get a permissions error when trying to
view information you do not have access to.

    ```
    dce accounts list
    err:  [GET /accounts][403] getAccountsForbidden
    ```

1. You will need to be authenticated as an admin before continuing to the next section. Type `dce auth` to log in as a different user. Sign out, then enter the username
and password for the admin that you created. As before, copy the auth code and paste it in the prompt in your command terminal.

    ![Admin login](./quickstartadminlogin.png)

1. Test that you have admin authorization by typing `dce accounts list`. You should see an empty list now instead of a permissions error.

    ```
    dce accounts list
    []
    ```

## Using IAM Credentials


The DCE API accepts authentication via IAM credentials using [SigV4 signed requests](https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html).

Any requests made via sufficiently permissioned IAM Credentials will be treated as an [admin role](#admins).

The process for signing requests with SigV4 is somewhat involved, but luckily there are a number of tools to make this easier. For example:

- [AWS Golang SDK `signer/v4` package](https://docs.aws.amazon.com/sdk-for-go/api/aws/signer/v4/)
- [aws-requests-auth](https://github.com/DavidMuller/aws-requests-auth) for Python
- [Postman _AWS Signature_ authentication](https://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-use-postman-to-call-api.html)

AWS also provides [examples for a number of languages in their docs](https://docs.aws.amazon.com/general/latest/gr/signature-v4-examples.html).

### Using IAM Credentials with the DCE CLI

The DCE CLI will search for credentials in the following order:

1. An API Token in the `api.token` field of the `.dce.yaml` config file. You may obtain an API Token by:
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

### IAM Policy for DCE API requests

The IAM principal used to send requests to the DCE API must have sufficient permissions to execute API requests.

The Terraform module in the repo provides an IAM policy with appropriate permissions for executing DCE API requests. The policy name and ARN are available as Terraform outputs.

```
terraform output api_access_policy_name
terraform output api_access_policy_arn
```