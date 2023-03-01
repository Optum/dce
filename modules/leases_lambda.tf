module "leases_lambda" {
  source          = "./lambda"
  name            = "leases-${var.namespace}"
  namespace       = var.namespace
  description     = "API /leases endpoints"
  global_tags     = var.global_tags
  handler         = "leases"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG                              = "false"
    NAMESPACE                          = var.namespace
    AWS_CURRENT_REGION                 = var.aws_region
    RESET_SQS_URL                      = aws_sqs_queue.account_reset.id
    ACCOUNT_DB                         = aws_dynamodb_table.accounts.id
    LEASE_DB                           = aws_dynamodb_table.leases.id
    LEASE_ADDED_TOPIC                  = aws_sns_topic.lease_added.arn
    DECOMMISSION_TOPIC                 = aws_sns_topic.lease_removed.arn
    COGNITO_USER_POOL_ID               = module.api_gateway_authorizer.user_pool_id
    COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME = var.cognito_roles_attribute_admin_name
    MAX_LEASE_BUDGET_AMOUNT            = var.max_lease_budget_amount
    MAX_LEASE_PERIOD                   = var.max_lease_period
    PRINCIPAL_BUDGET_AMOUNT            = var.principal_budget_amount
    PRINCIPAL_BUDGET_PERIOD            = var.principal_budget_period
    USAGE_CACHE_DB                     = aws_dynamodb_table.usage.id
  }
}

resource "aws_sns_topic" "lease_added" {
  name              = "lease-added-${var.namespace}"
  kms_master_key_id = local.sns_encryption_key_id
  tags              = var.global_tags
}

resource "aws_sns_topic" "lease_removed" {
  name              = "lease-removed-${var.namespace}"
  kms_master_key_id = local.sns_encryption_key_id
  tags              = var.global_tags
}


resource "aws_sns_topic" "lease_locked" {
  name              = "lease-locked-${var.namespace}"
  kms_master_key_id = local.sns_encryption_key_id
  tags              = var.global_tags
}

resource "aws_sns_topic" "lease_unlocked" {
  name              = "lease-unlocked-${var.namespace}"
  kms_master_key_id = local.sns_encryption_key_id
  tags              = var.global_tags
}
