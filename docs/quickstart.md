# Quickstart

Deploy DCE and lease an account quickly using the DCE CLI.

1. Download the appropriate executable for your OS from the [latest release](https://github.com/Optum/dce-cli/releases/latest). e.g. for mac, you should download dce_darwin_amd64.zip

1. Unzip and move the executable to a directory on your PATH, e.g.

    ```
    # Download the zip file
    curl -L -o dce_darwin_amd64.zip https://github.com/Optum/dce-cli/releases/latest/download/dce_darwin_amd64.zip

    # Unzip to a directory on your path
    unzip dce_darwin_amd64.zip -d /usr/local/bin
    ```

1. Type `dce init`. Leave all fields blank for now.

1. Deploy DCE using IAM Credentials that have AdministratorAccess
    ```
    export AWS_ACCESS_KEY_ID=XXXXXXXXXX
    export AWS_SECRET_ACCESS_KEY=XXXXXXXXXXXXXXXXXXXX
    dce system deploy
    ```

1. Retrieve the DCE API url from API Gateway in your master account, and add it to the dce config file, e.g.
    ```
    api:
        host: abcdefghij.execute-api.us-east-1.amazonaws.com
        basepath: /api
    region: us-east-1
    ```

1. Prepare a second AWS account to be your first "DCE Child Account".
    - Create an IAM role with `AdministratorAccess` and a trust relationship to your DCE Master Accounts
    - Create an account alias in the IAM dashboard or using the [AWS CLI command](https://docs.aws.amazon.com/cli/latest/reference/iam/create-account-alias.html)

    ```
    aws iam create-account-alias --account-alias examplealias
    ```

1. Add your child account to the accounts pool
    ```
    dce accounts add --account-id <child-account-id> --admin-role-arn <child-account-cross-account-role-arn>
    ```

1. Wait until the child account `accountStatus` is `Ready`
    ```
    dce accounts list
    [
        {
            "accountStatus": "Ready",
            "adminRoleArn": "arn:aws:iam::555555555555:role/DCEMasterAccess",
            "createdOn": 1575485630,
            "id": "775788068104",
            "lastModifiedOn": 1575485630,
            "principalPolicyHash": "\"bc5872b50475b186afea67ff47516a8f\"",
            "principalRoleArn": "arn:aws:iam::775788768154:role/DCEPrincipal-quickstart"
        }
    ]
    ```

1. Lease your child account
    ```
    dce leases create --budget-amount 100.0 --budget-currency USD --email jane.doe@email.com --principal-id quickstartuser
    ```

1. Log in to your leased account using the `--open-browser` flag to open the AWS Console in your default web browser. See the `howto guide <./howto.html#logging-into-a-leased-account>`_ for more login options.
    ```
    dce leases login --open-browser <lease-id>
    ```