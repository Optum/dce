# Concepts

## DCE

_Disposable Cloud Environment (DCE)_ provide temporary, limited access to Amazon Web 
Services (AWS) accounts. Administrators can configure this limited access to expire based on time or budget. When the access expires, DCE destroys all of the resources in the account and returns the account to the [account pool](#account-pool).

## Account
An _account_ is an AWS account that is available for leasing.

## Lease
A _lease_ is temporary access to an AWS account. A lease has a budget,
an expiration date, and a principal user.

## Reset

DCE _resets_ a leased child account during _any one_ of the following conditions:

* The time set on the `expiresOn` field is now in the past
* The amount set on the `budgetAmount` field is exceeded
* With a `/leases` API call or CLI command

To reset an account, DCE performs the following actions, in order:

1. Marks the lease as Inactive
1. Marks the account as Not Ready
1. Deletes all of the resources in the account.
1. Marks the account as Ready


## Account Status
The _account status_ indicates if the account is ready to be leased, leased
already, or in the process of being prepared to be leased again.

### Ready
An account in _Ready_ status is available for leasing. All of the resources
in the account have been cleaned and the account is like a brand-new, fresh
AWS account with the exception of an IAM role.

### Not Ready
An account in _Not Ready_ status is in the process of being reset
so that it can be marked as Ready. 

### Leased
An account in _Leased_ status is currently in use. A lease means
that the account is "checked out", much like a library book, a rental car, 
or a hotel room. 

## Lease Status

The _lease status_ indicates whether or not a lease is currently in use.

### Active
An _active_ lease is currently in use by the [principal](#principal) associated 
with the lease.

### Inactive
An _inactive_ lease is a lease that has either expired or the usage in the 
leased account has exceeded the budget on the lease.

## Lease Status Reason

### Expired

A lease that is _expired_ has exceeded the time set by the `expiresOn` field
of the lease. 

The API accepts a `expiresOn` field during lease creation.
DCE uses a configurable default read from the `DEFAULT_LEASE_LENGTH_IN_DAYS` environment variable when the `expiresOn` field is not present. If 
the configurable default is unset, DCE uses a period of seven (7) days.

### OverBudget

A lease that is _over budget_ has exceeded the budget amount set
by the `budgetAmount` field of the lease.

Each lease has a configurable [budget](#budget). DCE periodically
monitors the leased child accounts to determine when usage exceeds 
the budget amount queues the account for [reset](#reset).

### Destroyed

A lease may be destroyed before it expires or exceeds budget through 
the API or CLI. In this case, the lease status is marked "Inactive" and the 
reason is "Destroyed". The account associated with the lease is then
reset and the account is returned to the account pool.

### Active

A lease with an _Active_ status reason is an active lease.

### Rollback

A lease with the _Rollback_ lease status reason has experienced a failure
while DCE was getting the child account ready from the account pool. In the
event of a failure, DCE sets the lease status to _Inactive_ and the reason 
to _Rollback_ and returns the child account to the child pool.

## Account Pool

The _account pool_ is the collection of [_child accounts_](#child-account) that
are available for leasing.

## Master Account

The _master account_ is the AWS account that contains the DCE infrastructure
used to manage the child accounts that are in the account pool. 

## Child account

A _child account_ is an AWS account added to the account pool and 
controlled by the infrastructure in the master account. 

DCE requires an IAM role and permissions permissions to assume the role from 
the master account to control resources in the child account

## Principal

The _principal_ is the user to whom a child account is leased.

## Admin

The _admin_ is the user responsible for administering DCE.

## Admin Role

The _admin role_ is the role in the master account assumed by DCE to obtain
access to all resources in both the master and the child accounts.

## Principal Role

The _principal role_ is the IAM role in the child account that the 
principal assumes in order to access the resources in the account.

## Budget

A _budget_ is the amount of maximum spending that should be incurred during the lease. 
If the usage in the account exceeds the budget amount, DCE [resets](#reset) the 
account. 

## Usage

In DCE, _usage_ refers to the cost of running AWS resources in the accounts. 
