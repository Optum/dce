module "leases_lambda" {
  source          = "./lambda"
  name            = "leases-${var.namespace}"
  namespace       = var.namespace
  description     = "API /leases endpoints"
  global_tags     = var.global_tags
  handler         = "leases"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    RESET_SQS_URL      = aws_sqs_queue.account_reset.id
    ACCOUNT_DB         = aws_dynamodb_table.redbox_account.id
    LEASE_DB           = aws_dynamodb_table.redbox_lease.id
    PROVISION_TOPIC    = aws_sns_topic.lease_added.arn
    DECOMMISSION_TOPIC = aws_sns_topic.lease_removed.arn
  }
}

resource "aws_sns_topic" "lease_added" {
  name = "lease-added-${var.namespace}"
  tags = var.global_tags
}

resource "aws_sns_topic" "lease_removed" {
  name = "lease-removed-${var.namespace}"
  tags = var.global_tags
}


resource "aws_sns_topic" "lease_locked" {
  name = "lease-locked-${var.namespace}"
  tags = var.global_tags
}

resource "aws_sns_topic" "lease_unlocked" {
  name = "lease-unlocked-${var.namespace}"
  tags = var.global_tags
}
