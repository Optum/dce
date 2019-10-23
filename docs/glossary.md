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
### ...
## Account Pool

The _account pool_ is the group of _[child accounts](#child-accounts)_ that
are available for leasing.

## Master Account

The _master account_ is the AWS account that contains the DCE infrastructure
used to manage the child accounts that are in the account pool.

## Child account



## Principal
## Admin
## Admin Role
## Principal Role
## Budget
## Allowance <or whatever we're calling our per-principal budgets, I'm not crazy about "Allowance">
## Usage

