# Disposable Cloud Environment (DCE) 

## Using the `dce` command

The `dce` command allows you to provision DCE to your master account and 
administer the account. It also allows the leased account users to execute
commands for their accounts.

## Usage

To view the usage, run `dce` without any arguments are use the `help`
command as shown here:

```bash
$ dce help
$ dce account show [--usage|--status|--all (default)] 
$ dce account close
$ dce account request
$ dce account show-credentials
```

## Administrative commands

All administrative commands can be seen by using the 

```bash
$ dce help admin
$ dce admin init # Sets up DCE in the account
$ dce admin status # Like brew doctor, etc.

$ dce admin account add --account-id <account id>
$ dce admin account get-login --account-id <account id> # Gets a
$ dce admin account status --account-id <account id> # Gets a
$ dce admin account remove --account-id <account id>

$ dce admin lease list [--leased|--expired]
$ dce admin lease expire --lease-id [--expires-at <time>]
$ dce admin lease show-audits --lease-id <lease id>
# 
$ dce admin lease list-requests # Is there the notion of "auto approvals"? And if not, are there commands 
$ dce admin lease approve-request --least-request-id
$ dce admin lease set-approver [--auto|--email email-address
```