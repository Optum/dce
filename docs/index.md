# Disposable Cloud Environment (DCE)

The Disposable Cloud Environment (DCE) provide temporary,
limited Amazon Web Services (AWS) accounts. Accounts can be "leased" for a
period of time or up to a pre-determined budget amount. When the period of
time is reached or the maximum budgeted amount is exceeded, the lease is
expired. The leased account is _reset_ and returned to a pool of accounts
to be leased again.

At a high-level, DCE consists of [AWS Lambda](https://aws.amazon.com/lambda/) functions (implemented in [Go](https://golang.org/)), 
[Amazon DynamoDB](https://aws.amazon.com/dynamodb/) tables, 
[Amazon Simple Notifcation Servce (SNS)](https://aws.amazon.com/sns/) topics,
and APIs exposed with [Amazon API Gateway](https://aws.amazon.com/api-gateway/). 
These resources are created in the AWS account using [Hashicorp Terraform](https://www.terraform.io/).

## Why DCE?

> **Disposable Cloud Environment (DCE) are a playground in the cloud.**

With a DCE account, you have a safe environment to experiment with in the
cloud. With near-administrative access to an AWS account, you
can click around the AWS web console or run AWS CLI commands from your terminal. 
As a developer in an organization, you don't need to worry about cost management, 
orphaned resources, or lengthy cloud intake processes.

### Budget limits

DCE can be configured to expire account _leases_ (see [Glossary](/glossary/)) 
based on a budgeted amount for usage. 

Once the account hits a weekly spending limit, the account will be automatically
wiped clean so there are no surprise bills at the end of the month.

### Timed leases

DCE contains the concept of _expiring leases_. A account _lease_ is temporary
usage of an account for a certain amount of time. Once the lease is done,
the AWS account is cleaned of all of the resources and returned to the pool
for other people to lease.

## Getting started

To get started using DCE, see the [Quickstart](/quickstart/).

## Viewing the source

The source code for DCE can be found on [GitHub](https://github.com/Optum/dce).
