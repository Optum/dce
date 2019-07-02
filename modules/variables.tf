variable "aws_region" {
  description = "The AWS region for this Terraform run"
  default     = "us-east-1"
}

variable "global_tags" {
  description = "The tags to apply to all resources that support tags"
  type        = map(string)

  default = {
    Terraform = "True"
    Project   = "AWS Redbox Management"
    Source    = "github.com/Optum/Redbox//modules"
  }
}

variable "namespace" {
  description = "The namespace for this Terraform run"
}

variable "organization_id" {
  description = "The AWS Orgnanization ID the AWS Account is under."
  default     = "STUB"
}

variable "reset_account_admin_role" {
  description = "Default Admin IAM Role in each user account, that the Redbox Mgmt will assume in order to execute reset. Will get filtered from nuke."
  default     = "STUB"
}

variable "reset_account_user_role" {
  description = "Default User IAM Role in each user account, that the Redbox User will assume into their account. Will get filtered from nuke."
  default     = "STUB"
}

variable "reset_nuke_template_default" {
  description = "YAML file name of the default nuke configuration template."
  default     = "default-nuke-config-template.yml"
}

variable "reset_nuke_template_bucket" {
  description = "S3 bucket name containing the nuke configuration template."
  default     = "STUB"
}

variable "reset_nuke_template_key" {
  description = "S3 bucket object key for the nuke configuration template."
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
  default     = "true"
}

variable "reset_launchpad_toggle" {
  description = "Indicator to set Launchpad to not be applied. Use 'false' to indicate a Dry Run. NOTE: Cannot change Account status with this toggled off."
  default     = "true"
}

variable "reset_launchpad_base_endpoint" {
  description = "The Base URL of the Launchpad API Endpoint to apply Launchpad to an Account after a Reset."
  default     = "STUB"
}

variable "reset_launchpad_auth_endpoint" {
  description = "The URL of to retrieve an OAUTH token to be used with the Launchpad API."
  default     = "STUB"
}

variable "reset_launchpad_master_account" {
  description = "The Master Account Name of the AWS Account to reapply Launchpad under."
  default     = "POC"
}

variable "reset_launchpad_backend" {
  description = "The S3 Bucket name that contains the Terraform State files for the Account to be reset. These files will be remoeved."
  default     = "STUB"
}

variable "weekly_reset_cron_expression" {
  description = "The Weekly Reset Cron Expression used with CloudWatch to enqueue accounts for Reset."
  default     = "cron(0 5 ? * SUN *)" // Runs 5am GMT on Sunday / 12am on Sunday Morning
}

