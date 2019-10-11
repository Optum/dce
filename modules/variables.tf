variable "aws_region" {
  description = "The AWS region for this Terraform run"
  default     = "us-east-1"
}

variable "global_tags" {
  description = "The tags to apply to all resources that support tags"
  type        = map(string)

  default = {
    Terraform = "True"
    AppName   = "AWS Redbox Management"
    Source    = "github.com/Optum/Redbox//modules"
    Contact   = "CommercialCloudRedboxTeam_DL@ds.uhc.com"
  }
}

variable "namespace" {
  description = "The namespace for this Terraform run"
}

variable "organization_id" {
  description = "The AWS Orgnanization ID the AWS Account is under."
  default     = "STUB"
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
  description = "Indicator to set Nuke to not delete any resources. Use 'false' to indicate a Dry Run. NOTE: Cannot change Account status with this toggled off."
  default     = "false"
}

variable "weekly_reset_cron_expression" {
  description = "The Weekly Reset Cron Expression used with CloudWatch to enqueue accounts for Reset."
  default     = "cron(0 5 ? * SUN *)" // Runs 5am GMT on Sunday / 12am on Sunday Morning
}

variable "principal_iam_deny_tags" {
  type        = list(string)
  description = "IAM Redbox principal roles will be denied access to resources with these tags leased"
  default     = ["Redbox"]
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
  type = string
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
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of $${{.Lease.BudgetAmount}}. Actual spend is $${{.ActualSpend}}
{{else}}
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
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
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of $${{.Lease.BudgetAmount}}. Actual spend is $${{.ActualSpend}}
{{else}}
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of $${{.Lease.BudgetAmount}}.
Actual spend is $${{.ActualSpend}}
{{end}}
TMPL
}

variable "budget_notification_template_subject" {
  type        = string
  description = "Template for budget notification email subject"
  default     = <<SUBJ
AWS Redbox Lease {{if .IsOverBudget}}over budget{{else}}at {{.ThresholdPercentile}}% of budget{{end}} [{{.Lease.AccountID}}]
SUBJ
}

variable "budget_notification_threshold_percentiles" {
  type        = list(number)
  description = "Thresholds (percentiles) at which notification emails will be sent to Redbox users."
  default     = [75, 100]
}

variable "redbox_principal_policy" {
  type        = string
  description = "Location of file with the policy used for the RedBox Principal Account"
  default     = ""
}

variable "update_lease_status_schedule_expression" {
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
  default = "DCE"
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
