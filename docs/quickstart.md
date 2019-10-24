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

## Basic steps

To deploy and start using DCE, follow these basic steps (explained in 
greater detail below):

1. Deploy DCE to your master account.
1. Provision the IAM role in the child account.
1. Add each account to the account pool by using the 
[CLI](/using-the-cli/) or [REST API](/api-documentation/).

Each of these steps is covered in greater detail in the sections below.

### Deploying DCE to the master account

To deploy DCE into the "master" account, you will need to start out with
an existing AWS account. The `dce` command line interface (CLI) is the easiest
way to deploy DCE into your master account.

### Provisioning the IAM role in the child account



### Adding child accounts to the account pool

