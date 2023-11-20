module "update_principal_policy" {
  source          = "./lambda"
  name            = "update_principal_policy-${var.namespace}"
  namespace       = var.namespace
  description     = "Updates the Principal Policy"
  global_tags     = var.global_tags
  handler         = "update_principal_policy"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn
  dlq_enabled     = true

  environment = {
    DEBUG                          = "false"
    NAMESPACE                      = var.namespace
    AWS_CURRENT_REGION             = var.aws_region
    ACCOUNT_DB                     = aws_dynamodb_table.accounts.id
    LEASE_DB                       = aws_dynamodb_table.leases.id
    ARTIFACTS_BUCKET               = aws_s3_bucket.artifacts.id
    PRINCIPAL_ROLE_NAME            = local.principal_role_name
    PRINCIPAL_POLICY_NAME          = local.principal_policy_name
    PRINCIPAL_POLICY_S3_KEY        = aws_s3_object.principal_policy.key
    PRINCIPAL_IAM_DENY_TAGS        = join(",", var.principal_iam_deny_tags)
    ALLOWED_REGIONS                = join(",", var.allowed_regions)
    PRINCIPAL_MAX_SESSION_DURATION = 14400
    TAG_ENVIRONMENT                = var.namespace == "prod" ? "PROD" : "NON-PROD"
    TAG_APP_NAME                   = lookup(var.global_tags, "AppName")
  }
}

resource "aws_sns_topic_subscription" "update_principal_policy" {
  topic_arn = aws_sns_topic.lease_unlocked.arn
  protocol  = "lambda"
  endpoint  = module.update_principal_policy.arn
}

resource "aws_lambda_permission" "update_principal_policy" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = module.update_principal_policy.name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.lease_unlocked.arn
}

resource "aws_sns_topic_subscription" "update_principal_policy_on_lease_create" {
  topic_arn = aws_sns_topic.lease_added.arn
  protocol  = "lambda"
  endpoint  = module.update_principal_policy.arn
}

resource "aws_lambda_permission" "update_principal_policy_on_lease_create" {
  statement_id  = "AllowInvokeFromLeaseAddedTopic"
  action        = "lambda:InvokeFunction"
  function_name = module.update_principal_policy.name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.lease_added.arn
}

resource "aws_iam_role_policy" "update_principal_policy" {
  role   = module.update_principal_policy.execution_role_name
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
        "Effect": "Allow",
        "Action": [
            "dynamodb:GetItem"
        ],
        "Resource": "${aws_dynamodb_table.accounts.arn}"
    },
    {
        "Effect": "Allow",
        "Action": [
            "sts:AssumeRole"
        ],
        "Resource": "*"
    }
  ]
}
POLICY
}
