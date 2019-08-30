module "update-redbox-principal-policy" {
  source          = "./lambda"
  name            = "update_redbox_principal_policy-${var.namespace}"
  namespace       = var.namespace
  description     = "Reset the Redbox Principal Policy"
  global_tags     = var.global_tags
  handler         = "update_redbox_principal_policy"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG                          = "false"
    NAMESPACE                      = var.namespace
    AWS_CURRENT_REGION             = var.aws_region
    ACCOUNT_DB                     = aws_dynamodb_table.redbox_account.id
    LEASE_DB                       = aws_dynamodb_table.redbox_lease.id
    ARTIFACTS_BUCKET               = aws_s3_bucket.artifacts.id
    PRINCIPAL_ROLE_NAME            = local.redbox_principal_role_name
    PRINCIPAL_POLICY_NAME          = local.redbox_principal_policy_name
    PRINCIPAL_POLICY_S3_KEY        = aws_s3_bucket_object.redbox_principal_policy.key
    PRINCIPAL_IAM_DENY_TAGS        = join(",", var.principal_iam_deny_tags)
    PRINCIPAL_MAX_SESSION_DURATION = 14400
    TAG_ENVIRONMENT                = var.namespace == "prod" ? "PROD" : "NON-PROD"
    TAG_CONTACT                    = lookup(var.global_tags, "Contact")
    TAG_APP_NAME                   = lookup(var.global_tags, "AppName")
  }
}

resource "aws_sns_topic_subscription" "update-redbox-principal-policy" {
  topic_arn = aws_sns_topic.lease_unlocked.arn
  protocol  = "lambda"
  endpoint  = module.update-redbox-principal-policy.arn
}

resource "aws_lambda_permission" "update-redbox-principal-policy" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = module.update-redbox-principal-policy.name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.lease_unlocked.arn
}

resource "aws_iam_role_policy" "update-redbox-principal-policy" {
  role   = module.update-redbox-principal-policy.execution_role_name
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
        "Effect": "Allow",
        "Action": [
            "dynamodb:GetItem"
        ],
        "Resource": "${aws_dynamodb_table.redbox_account.arn}"
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
