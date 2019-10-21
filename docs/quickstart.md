# DCE Quickstart

The purpose of this quickstart is to show how to deploy DCE into
a _master account_ and to show you how to add other AWS accounts
into the _lease pool_.

Before you begin, it is helpful to understand a few concepts.

* **master account**: In DCE, the _master account_ is the account
in which the infrastructure is deployed that handles provisioning
leases, keeping track of who has the leases, keeping track of which
AWS accounts are in the lease pool, and running opertions on the 
lease accounts to clean them up and return them to the lease pool.
* **lease pool**: The _lease pool_ is a logical group of AWS accounts
that are available for leasing.


## Prerequisites

Before you can deploy and use DCE, you will need the following:

1. An AWS account for the master account, and **sufficient credentials**
for deploying DCE into the account.
1. One or more AWS account to add to the lease pool. As of the time of this writing, 
DCE does not _create_ any AWS accounts for you. You will need to bring your own 
AWS accounts for adding to the lease pool.
1. In each account you add to the lease pool, you will create an IAM role
that allows DCE to control the leased account. This is detailed later 
in this quickstart.

## Basic steps

To deploy and start using DCE, follow these basic steps (explained in 
greater detail below):

1. Deploy DCE to your master account.
1. Provision the IAM role in the lease account.
1. Add each lease account to the lease pool by using the 
[CLI](/using-the-cli/) or [REST API](/api-documentation/).

Each of these steps is covered in greater detail in the sections below.

### Deploying DCE to the master account

To deploy DCE into the "master" account, you will need to start out with
an existing AWS account. The