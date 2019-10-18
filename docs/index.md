# Disposable Cloud Environments (DCE)

The Disposable Cloud Environments (DCE) is a mechanism for providing temporary, limited Amazon Web Services (AWS)
accounts. Accounts can be "leased" for a period of seven days (by default). After the time has expired, the 
account is _reset_ and returned to a pool of accounts to be leased again.

At a high-level, DCE consists of [AWS Lambda](https://aws.amazon.com/lambda/) functions (implemented in [Go](https://golang.org/)), 
[Amazon DynamoDB](https://aws.amazon.com/dynamodb/) tables, 
[Amazon Simple Notifcation Servce (SNS)](https://aws.amazon.com/sns/) topics,
and APIs exposed with [Amazon API Gateway](https://aws.amazon.com/api-gateway/). 
These resources are created in the AWS account using [Hashicorp Terraform](https://www.terraform.io/).

## Getting started

To get started using DCE, see the [Quickstart](/quickstart).

## Project layout

    mkdocs.yml    # The configuration file.
    docs/
        index.md  # The documentation homepage.
        ...       # Other markdown pages, images and other files.
