# Glossary

## DCE

_Disposable Cloud Environments (DCE)_ provide temporary, limited Amazon Web 
Services (AWS) accounts. Accounts can expire based on time or budget. Upon
expiration, all of the resources in the account are destroyed and the account
is returned to the [account pool](#account-pool)

## Account
An _account_ is an AWS account that is available for leasing.

## Lease
A _lease_ is temporary access to an AWS account.

## Account Status
The _account status_ is the status of the account in the 
[account pool](#account-pool).

### Ready
An account in _Ready_ status is available for leasing. All of the resources
in the account have been cleaned and the account is like a brand-new, fresh
AWS account with the exception of an IAM role.

### Not Ready
An account in _Not Ready_ (`NonReady`) status is in the process of being reset
so that it can be marked as Ready. 

### Leased
An account that is in _Leased_ status is currently in use. A lease means
that the account is "checked out", much like a library card, a rental car, 
or a hotel room. 

## Lease Status

The _lease status_ indicates the availability of a lease.

### Active
When a lease is _active_, the associated account is currently in use by the
[principal] associated with the lease, either the amount of time until the 
lease's expiration date or until the lease's budget has been exceeded.

### Inactive
An _inactive_ lease is a lease that has either expired or the usage in the 
account exceeded the budget on the lease.

## Lease Status Reason

### Expired

A lease that is _expired_ has exceeded the time set by the `expiresOn` field
of the lease. 

The `expiresOn` field may be set during lease creation with the 
API. If not set, a configurable default will be used. If this configurable
default is not set (with the environment variable `TODO`), DCE uses a period of 
seven (7) days.

### OverBudget

Each lease has a configurable budget (see "Budget"). When the budget is 
exceeded, DCE resets the account and returns the child account to the 
account pool with the status of "Ready".

### Destroyed

A lease may be destroyed before it expires or exceeds budget using the API
or CLI. In this case, the lease status is marked "Inactive" and the reason
is "Destroyed".

### Active



## Account Pool

The _account pool_ is the group of _[child accounts](#child-accounts)_ that
are available for leasing.

## Master Account

The _master account_ is the AWS account that contains the DCE infrastructure
used to manage the child accounts that are in the account pool. 

## Child account

A _child account_ is an AWS account that is added to the account pool and is
controlled by the infrastructure in the master account. 

In order for the master account to control resources in the child account, an 
IAM role is created and permission granted for the DCE role in the master 
account to assume the role.

## Principal

The _principal_ is the name of the main user of an account.

## Admin


## Admin Role

The _admin role_ is the role in the master account assumed by DCE to obtain
access to resources in both the master and the child accounts.

## Principal Role

The _principal role_ is the IAM role in the child account that DCE will assume
in order to do its work, such as reseting the resources in the account.

## Budget

A _budget_ is a fixed amount of spending that can be associated with an account
lease. Once the amount in the budget has been exceeded, DCE will reset the 
account by deleting all of the resources in the account. Once the resources have
been deleted, the account will be added back to the account pool and marked as
_Ready_ so that it may be leased again.

## Allowance <or whatever we're calling our per-principal budgets, I'm not crazy about "Allowance">
## Usage

In DCE, _usage_ refers to the cost of running AWS resources in the accounts. 
