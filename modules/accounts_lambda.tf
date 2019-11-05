locals {
  principal_role_name   = "${var.namespace_prefix}Principal${var.namespace == "prod" ? "" : "-${var.namespace}"}"
  principal_policy_name = "${var.namespace_prefix}PrincipalDefaultPolicy${var.namespace == "prod" ? "" : "-${var.namespace}"}"
}

module "accounts_lambda" {
  source           = "./lambda"
  name             = "accounts"
  namespace_prefix = var.namespace_prefix
  namespace        = var.namespace
  description      = "Handles API requests to the /accounts endpoint"
  global_tags      = var.global_tags
  handler          = "accounts"
  alarm_topic_arn  = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG                          = "false"
    NAMESPACE                      = var.namespace
    AWS_CURRENT_REGION             = var.aws_region
    ACCOUNT_DB                     = aws_dynamodb_table.accounts.id
    ARTIFACTS_BUCKET               = aws_s3_bucket.artifacts.id
    LEASE_DB                       = aws_dynamodb_table.leases.id
    RESET_SQS_URL                  = aws_sqs_queue.account_reset.id
    ACCOUNT_CREATED_TOPIC_ARN      = aws_sns_topic.account_created.arn
    ACCOUNT_DELETED_TOPIC_ARN      = aws_sns_topic.account_deleted.arn
    PRINCIPAL_ROLE_NAME            = local.principal_role_name
    PRINCIPAL_POLICY_NAME          = local.principal_policy_name
    PRINCIPAL_IAM_DENY_TAGS        = join(",", var.principal_iam_deny_tags)
    PRINCIPAL_MAX_SESSION_DURATION = 14400
    TAG_ENVIRONMENT                = var.namespace == "prod" ? "PROD" : "NON-PROD"
    TAG_CONTACT                    = lookup(var.global_tags, "Contact")
    TAG_APP_NAME                   = lookup(var.global_tags, "AppName")
    PRINCIPAL_POLICY_S3_KEY        = aws_s3_bucket_object.principal_policy.key
  }
}

resource "aws_sns_topic" "account_created" {
  name = "${var.namespace_prefix}-account-created-${var.namespace}"
  tags = var.global_tags
}

resource "aws_sns_topic" "account_deleted" {
  name = "${var.namespace_prefix}-account-deleted-${var.namespace}"
  tags = var.global_tags
}
