module "fan_out_check_budget_lambda" {
  source          = "./lambda"
  name            = "fan_out_check_budget-${var.namespace}"
  namespace       = var.namespace
  description     = "Initiates the budget check lambda. Invokes a check-budget lamdba for each active lease"
  global_tags     = var.global_tags
  handler         = "fan_out_check_budget"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    AWS_CURRENT_REGION         = var.aws_region
    ACCOUNT_DB                 = aws_dynamodb_table.dce_account.id
    LEASE_DB                   = aws_dynamodb_table.dce_lease.id
    CHECK_BUDGET_FUNCTION_NAME = module.check_budget_lambda.name
  }
}

// Allow fan_out_check_budget to invoke the check_budget lambda
resource "aws_iam_role_policy_attachment" "fan_out_check_budget_invoke_lambda" {
  role       = module.fan_out_check_budget_lambda.execution_role_name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaRole"
}

module "check_budget_lambda" {
  source          = "./lambda"
  name            = "check_budget-${var.namespace}"
  namespace       = var.namespace
  description     = "Checks spend for a lease within an AWS account, and locks lease if over budget"
  handler         = "check_budget"
  global_tags     = var.global_tags
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    AWS_CURRENT_REGION                        = var.aws_region
    ACCOUNT_DB                                = aws_dynamodb_table.dce_account.id
    LEASE_DB                                  = aws_dynamodb_table.dce_lease.id
    USAGE_CACHE_DB                            = aws_dynamodb_table.usage_cache.id
    RESET_QUEUE_URL                           = aws_sqs_queue.account_reset.id
    LEASE_LOCKED_TOPIC_ARN                    = aws_sns_topic.lease_locked.arn
    BUDGET_NOTIFICATION_FROM_EMAIL            = var.budget_notification_from_email
    BUDGET_NOTIFICATION_BCC_EMAILS            = join(",", var.budget_notification_bcc_emails)
    BUDGET_NOTIFICATION_TEMPLATE_HTML         = var.budget_notification_template_html
    BUDGET_NOTIFICATION_TEMPLATE_TEXT         = var.budget_notification_template_text
    BUDGET_NOTIFICATION_TEMPLATE_SUBJECT      = var.budget_notification_template_subject
    BUDGET_NOTIFICATION_THRESHOLD_PERCENTILES = join(",", var.budget_notification_threshold_percentiles)
  }
}

// Allow check_budget lambda to send emails with SES
resource "aws_iam_role_policy" "check_buget_ses" {
  role   = module.check_budget_lambda.execution_role_name
  policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": ["ses:SendEmail"],
      "Resource": "*"
    }]
}
POLICY
}

// Run the fan-out-check-budget lambda on a timer (cloudwatch event)
module "dbbackup_lambda_schedule" {
  source              = "./cloudwatch_event"
  name                = "fan_out_check_budget-${var.namespace}"
  lambda_function_arn = module.fan_out_check_budget_lambda.arn
  schedule_expression = var.check_budget_schedule_expression
  description         = "Initiates the budget check lambda. Invokes a check-budget lamdba for each active lease"
  enabled             = var.check_budget_enabled
}
