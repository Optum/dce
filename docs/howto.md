# How To

A practical guide to common operations and customizations for DCE.

## Use the DCE API

DCE provides a set of endpoints for managing account pools and leases, and for monitoring account usage. 

See [API Reference Documentation](./api-documentation.md) for details.

See [API Auth Documentation](./api-auth.md) for details on authenticating and authorizing requests. 

### API Location

The API is hosted by AWS API Gateway. The base URL is exposed as a Terraform output. API Gateway generates a unique ID as part of the API URL. To retrieve the base url of the API, run the following command from [the Terraform modules directory](https://github.com/Optum/dce/tree/master/modules):

```
terraform output api_url
```

All endpoints use this value as the base url. For example, to view accounts:

```
GET https://asdfghjkl.execute-api.us-east-1.amazonaws.com/api/accounts
``` 

## Use the DCE CLI

DCE provides a CLI tool to deploy DCE, interact with DCE APIs, and login to DCE child accounts. For example:

```bash
# Deploy DCE
dce system deploy

# Add an account to the pool
dce accounts add \
    --account-id 123456789012 \
    --admin-role-arn arn:aws:iam::123456789012:role/OrganizationAccountAccessRole

# Lease an account
dce leases create \
    --principal-id jdoe@example.com \
    --budget-amount 100 --budget-currency USD

# Login to your account
dce leases login <lease-id>
```

See the [github.com/Optum/dce-cli](https://github.com/Optum/dce-cli) repo for details.

## Login to your DCE Account

The easiest way for users to login to their DCE child account is via the [DCE CLI](https://github.com/Optum/dce-cli):

```
dce leases login <lease-id>
```

This command generates temporary CLI credentials for the AWS child account, and saves them to the user's AWS CLI configuration.

See the [DCE CLI reference docs](https://github.com/Optum/dce-cli/blob/master/docs/dce_leases_login.md) for details.

## Add Accounts to the DCE Account Pool

DCE manages its collection of AWS accounts in an [account pool](concepts.md#account-pool). Each account in the pool is made available for [leasing](concepts.md#lease) by DCE users.

DCE _does not_ create AWS accounts. These must be added to the account pool by a DCE administrator. You can create accounts using the AWS [`CreateAccount` API](https://docs.aws.amazon.com/cli/latest/reference/organizations/create-account.html).

The child account must have an administrative IAM Role with a trust relationship to allow the master account to assume the role. For example:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::MASTER_ACCOUNT_ID:root"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

Add the account to the pool using the DCE API:

`POST /accounts`
```json
{
  "id": "<child_account_id>",
  "adminRoleArn": "<ARN of the admin IAM role in the child account>"
}
```

Or use the [DCE CLI](#use-the-dce-cli):

```
dce accounts add \
    --account-id <child_account_id> \
    --admin-role-arn <ARN of the admin IAM role in the child account>
```



## Configure Budgets and Lease Periods

Every [lease](concepts.md#lease) comes with a configured **per-lease budget**, which limits AWS account spend during the course of the lease. Additionally there are **per-principal budgets**, which limit spend by a single user across multiple lease during a budget period. This prevents a single user from creating multiple leases to as a way of circumventing lease budgets.

DCE budget may be configured as [Terraform variables](terraform.md#configuring-terraform-variables).

| Variable | Default | Description |
| --- | --- | --- |
| `max_lease_budget_amount` | 1000 | The maximum budget a user may request for their lease |
| `max_lease_period` | 604800 | The maximum duration (seconds) a user may request for their lease |
| `principal_budget_amount` | 1000 | The maximum spend a user may accumulate across any number of leases during the `principal_budget_period` |
| `principal_budget_period` | "WEEKLY" | The period across which the `principal_budget_amount` is measured. Currently only supports "WEEKLY" |


## Configure Account Resets

To [reset](concepts.md#reset) AWS accounts between leases, DCE uses the [open source `aws-nuke` tool](https://github.com/rebuy-de/aws-nuke). This tool attempts to delete every single resource in th AWS account, and will make several attempts to ensure everything is wiped clean.

To prevent `aws-nuke` from deleting certain resources, provide a YAML configuration with a list of resource _filters_. (see [`aws-nuke` docs for the YAML filter configuration syntax](https://github.com/rebuy-de/aws-nuke#filtering-resources)). By default, DCE filters out resources which are critical to running DCE -- for example, the IAM roles for your account's `adminRoleArn` / `principalRoleArn`.

As a DCE implementor, you may have additional resources you wish protect from `aws-nuke`. If this is the case, you may specify your own custom `aws-nuke` YAML configuration:

- Copy the contents of [`default-nuke-config-template.yml`](https://github.com/Optum/dce/blob/master/cmd/codebuild/reset/default-nuke-config-template.yml) into a new file
- Modify as needed.
- Upload the YAML configuration file to an S3 bucket in the DCE master account

Then configure reset using [Terraform variables](terraform.md#configuring-terraform-variables):

| Variable | Default | Description |
| --- | --- | --- |
| `reset_nuke_template_bucket` | See [`default-nuke-config-template.yml`](https://github.com/Optum/dce/blob/master/cmd/codebuild/reset/default-nuke-config-template.yml) | S3 bucket where a custom [aws-nuke](https://github.com/rebuy-de/aws-nuke) configuration is located |
| `reset_nuke_template_key` | See [`default-nuke-config-template.yml`](https://github.com/Optum/dce/blob/master/cmd/codebuild/reset/default-nuke-config-template.yml) | S3 key within the `reset_nuke_template_bucket` where a custom [aws-nuke](https://github.com/rebuy-de/aws-nuke) configuration is located |
| `reset_nuke_toggle` | `true` | Set to false to run `aws-nuke` in dry run mode |
| `allowed_regions` | _all AWS regions_ | AWS regions which will be nuked. Allowing fewer regions will drastically reduce the run time of aws-nuke | 


## Customize Budget Notifications

When a lease owner approaches or exceeds their budget, they will receive an email notification. These notifications are [configurable as Terraform variables](terraform.md#configuring-terraform-variables):

| Variable | Default | Description |
| --- | --- | --- |
| `check_budget_enabled` | `true` | Set to `false` to disable budget checks entirely |
| `budget_notification_threshold_percentiles` | `[75, 100]` | Thresholds (percentiles) at which budget notification emails will be sent to users. |
| `budget_notification_from_email` | `"dce@example.com"` | `FROM` email address for budget notifications |
| `budget_notification_bcc_emails` | `[]` | Budget notifications emails will be BCC'd to these addresses |
| `budget_notification_template_subject` | See [variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) | Template for budget notification email subject |
| `budget_notification_template_text` | See [variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) | Template for budget notification text emails |
| `budget_notification_template_html` | See [variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) | Template for budget notification HTML emails |


### Email Templates

Budget notification email templates are rendered using [golang templates](https://golang.org/pkg/text/template/), and accept the following arguments:

| Argument | Description |
| --- | --- |
| IsOverBudget | Set to `true` if the account is over the configured budget |
| Lease.PrincipalID | The principal ID of the lease holder |
| Lease.AccountID | The Account number of the AWS account in use |
| Lease.BudgetAmount | The configured budget amount for the lease |
| ActualSpend | The calculated spend on the account at time of notification |
| ThresholdPercentile | The configured threshold percentage for the notification |


## Backup DCE Database Tables

DCE does not backup DynamoDB tables by default. However, if you want to restore a DynamoDB table from a backup, we do provide a helper script in [scripts/restore_db.sh](https://github.com/Optum/dce/blob/master/scripts/restore_db.sh). This script is also provided as a Github release artifact, for easy access.

To restore a DynamoDB table from a backup:

```
# Grab the account table name from Terraform state
table_name=$(cd modules && terraform output accounts_table_name)

# Or, grab the leases table name
table_name=$(cd modules && terraform output leases_table_name)

# List available backups
./scripts/restore_db.sh \
  --target-table-name ${table_name} \
  --list-backups

# Choose an backup from the output of the last command, and pass in the ARN
./scripts/restore_db.sh \
  --target-table-name ${table_name} \
  --backup-arn <backup arn>

# If the table already exists, and you want to delete and
# recreate it from a backup, pass in
# the --force-delete-table flag
./scripts/restore_db.sh \
  --target-table-name ${table_name} \
  --backup-arn <backup arn> \
  --force-delete-table
```

After restoring the DynamoDB table from a backup, [re-apply Terraform](terraform.md#deploying-dce-with-terraform) to ensure that your table is in sync with your Terraform configuration.

See [AWS guide for backing up DynamoDB tables](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Backup.Tutorial.html).

## Monitor DCE

DCE comes prebuilt with a number of CloudWatch alarms, which will trigger when DCE systems encounter errors or behaves abnormally.

These CloudWatch alarms are delivered to an SNS topic. The ARN of this SNS topic is available as [a Terraform output](terraform.md#accessing-terraform-outputs):

```
cd modules
terraform output alarm_sns_topic_arn
```

Subscribe to this topic to receive alarm notifications. For example:

```bash
aws sns subscribe \
  --topic-arn <Alarm Topic ARN> \
  --protocol email \
  --notification-endpoint my-email@example.com
``` 