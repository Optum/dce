variable "aws_region" {
  description = "The AWS region for this Terraform run"
  default     = "us-east-1"
}

variable "account_pool_metrics_toggle" {
  description = "Set to 'true' to enable periodic scanning of the Accounts table for account status metrics. Defaults to 'false'"
  default     = "false"
}

variable "account_pool_metrics_collection_rate_expression" {
  description = "The rate at which the Accounts table will be scanned for Account status metrics. Defaults to rate(30 minutes). See https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html"
  default     = "rate(30 minutes)"
}

variable "account_pool_metrics_widget_period" {
  description = "The aggregation period used by the account pool metrics dashboard widget (in seconds). Must be less than the account_pool_metrics_collection_rate_expression. Defaults to 1200"
  default     = "1200"
}

variable "global_tags" {
  description = "The tags to apply to all resources that support tags"
  type        = map(string)

  default = {
    Terraform = "True"
    AppName   = "DCE"
  }
}

variable "namespace" {
  description = "The namespace for this Terraform run"
}

variable "reset_nuke_template_bucket" {
  description = "S3 bucket name containing the nuke configuration template. Use this to override the default nuke configuration."
  default     = "STUB"
}

variable "reset_nuke_template_key" {
  description = "S3 bucket object key for the nuke configuration template. Use this to override the default nuke configuration."
  default     = "STUB"
}

variable "reset_build_image" {
  description = "Docker image to run the Reset CodeBuild."
  default     = "aws/codebuild/standard:1.0"
}

variable "reset_compute_type" {
  description = "Compute type to run the Reset CodeBuild."
  default     = "BUILD_GENERAL1_SMALL"
}

variable "reset_build_type" {
  description = "Image build type to run the Reset CodeBuild."
  default     = "LINUX_CONTAINER"
}

variable "reset_image_pull_creds" {
  description = "Service or service role to pull the Docker image."
  default     = "CODEBUILD"
}

variable "reset_nuke_toggle" {
  description = "Use 'false' to indicate a disable aws-nuke, and avoid deleted any resources in child accounts."
  default     = "true"
}

variable "cloudwatch_dashboard_toggle" {
  description = "Set to 'true' to enable an out of the box cloudwatch dashboard. Defaults to 'false."
  default     = "false"
}

variable "populate_reset_queue_schedule_expression" {
  description = "The schedule used with CloudWatch to enqueue accounts for reset."
  default     = "rate(6 hours)" // Runs every six hours
}

variable "principal_iam_deny_tags" {
  type        = list(string)
  description = "IAM principal roles will be denied access to resources with the `AppName` tag set to this value"
  default     = ["DCE"]
}

variable "check_budget_schedule_expression" {
  default     = "rate(6 hours)"
  description = "How often to check budgets for all active leases"
}
variable "check_budget_enabled" {
  type        = bool
  default     = true
  description = "If false, budgets will not be checked"
}
variable "budget_notification_from_email" {
  type    = string
  default = "notifications@example.com"
}

variable "budget_notification_bcc_emails" {
  type        = list(string)
  description = "Budget notifications emails will be bcc-d to these addresses"
  default     = []
}

variable "budget_notification_template_html" {
  type        = string
  description = "HTML template for budget notification emails"
  default     = <<TMPL
<p>
{{if .IsOverBudget}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of $${{.Lease.BudgetAmount}}. Actual spend is $${{.ActualSpend}}
{{else}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of $${{.Lease.BudgetAmount}}.
Actual spend is $${{.ActualSpend}}
{{end}}
</p>
TMPL
}

variable "budget_notification_template_text" {
  type        = string
  description = "Text template for budget notification emails"
  default     = <<TMPL
{{if .IsOverBudget}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of $${{.Lease.BudgetAmount}}. Actual spend is $${{.ActualSpend}}
{{else}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of $${{.Lease.BudgetAmount}}.
Actual spend is $${{.ActualSpend}}
{{end}}
TMPL
}

variable "budget_notification_template_subject" {
  type        = string
  description = "Template for budget notification email subject"
  default     = <<SUBJ
Lease {{if .IsOverBudget}}over budget{{else}}at {{.ThresholdPercentile}}% of budget{{end}} [{{.Lease.AccountID}}]
SUBJ
}

variable "budget_notification_threshold_percentiles" {
  type        = list(number)
  description = "Thresholds (percentiles) at which budget notification emails will be sent to users."
  default     = [75, 100]
}

variable "principal_policy" {
  type        = string
  description = "Location of file with the policy to be attached to principal IAM users"
  default     = ""
}

variable "fan_out_update_lease_status_schedule_expression" {
  type        = string
  description = "Update lease status schedule"
  default     = "rate(6 hours)"
}

variable "update_lease_status_enabled" {
  type        = bool
  description = "Update lease status enabled"
  default     = true
}

variable "namespace_prefix" {
  type    = string
  default = "dce"
}

variable "cognito_callback_urls" {
  type    = list(string)
  default = ["https://127.0.0.1:8080/login"]
}

variable "cognito_logout_urls" {
  type    = list(string)
  default = ["https://127.0.0.1:8080/logout"]
}

variable "cognito_identity_providers" {
  type    = list(string)
  default = ["COGNITO"]
}

variable "cognito_roles_attribute_admin_name" {
  type    = string
  default = "Admin"
}

variable "max_lease_budget_amount" {
  type        = number
  description = "Lease budget amount for given lease budget period"
  default     = 1000
}

variable "max_lease_period" {
  type        = number
  description = "Lease's maximum period in seconds"
  default     = 604800
}

variable "principal_budget_amount" {
  type        = number
  description = "User Principal's budget amount for given principal budget period"
  default     = 1000
}

variable "principal_budget_period" {
  type        = string
  description = "Principal budget period must be WEEKLY or MONTHLY"
  default     = "WEEKLY"
}

variable "allowed_regions" {
  type = list(string)
  default = [
    "us-east-1"
  ]
  description = "List of AWS regions which DCE Principals have access to. These regions will also be targeted for reset in nuke.yml."
}

variable "orphaned_accounts_alarm_threshold" {
  type        = string
  description = "Alarm when number of orphaned accounts is greater than or equal to this threshold."
  default     = "1"
}

variable "ready_accounts_alarm_threshold" {
  type        = string
  description = "Alarm when number of ready accounts is less than or equal to this threshold."
  default     = "20"
}

variable "usage_ttl" {
  type = number
  # 30 days
  default     = 2592000
  description = "TTL in seconds for records in the Usage DynamoDB table. Records older than this TTL will be automatically deleted."
}

variable "accounts_table_rcu" {
  type        = number
  default     = 5
  description = "DynamoDB Accounts table provisioned Read Capacity Units (RCUs). See https://aws.amazon.com/dynamodb/pricing/provisioned/"
}

variable "accounts_table_wcu" {
  type        = number
  default     = 5
  description = "DynamoDB Accounts table provisioned Write Capacity Units (WCUs). See https://aws.amazon.com/dynamodb/pricing/provisioned/"
}

variable "leases_table_rcu" {
  type        = number
  default     = 5
  description = "DynamoDB Leases table provisioned Read Capacity Units (RCUs). See https://aws.amazon.com/dynamodb/pricing/provisioned/"
}

variable "leases_table_wcu" {
  type        = number
  default     = 5
  description = "DynamoDB Leases table provisioned Write Capacity Units (WCUs). See https://aws.amazon.com/dynamodb/pricing/provisioned/"
}

variable "usage_table_rcu" {
  type        = number
  default     = 5
  description = "DynamoDB Usage table provisioned Read Capacity Units (RCUs). See https://aws.amazon.com/dynamodb/pricing/provisioned/"
}

variable "usage_table_wcu" {
  type        = number
  default     = 5
  description = "DynamoDB Usage table provisioned Write Capacity Units (WCUs). See https://aws.amazon.com/dynamodb/pricing/provisioned/"
}

variable "cloudwatch_log_retention" {
  type        = number
  default     = 1
  description = "CloudWatch Log retention in days"
}