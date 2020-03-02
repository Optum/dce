
data "aws_iam_policy_document" "state_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["states.${var.aws_region}.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "state" {
  name               = "state-exec-${var.namespace}"
  assume_role_policy = data.aws_iam_policy_document.state_assume_role.json
}

data "aws_iam_policy_document" "lambda-invoke" {
  statement {
    actions = [
      "lambda:InvokeFunction"
    ]
    resources = [
      "*",
    ]
  }
}

resource "aws_iam_role_policy" "state_invoke_lame" {
  name   = "InvokeLambda"
  role   = aws_iam_role.state.id
  policy = data.aws_iam_policy_document.lambda-invoke.json
}

resource "aws_sfn_state_machine" "lease_usage" {
  name     = "lease-usage-${var.namespace}"
  role_arn = aws_iam_role.state.arn

  definition = templatefile("${path.module}/fixtures/state_machines/lease.json", {
    WAIT_SECONDS         = var.lease_state_machine_wait_seconds
    GET_USAGE_LAMBDA_ARN = module.state_get_lease_usage_lambda.arn
    GET_LEASE_LAMBDA_ARN = module.state_get_lease_lambda.arn
    END_LEASE_LAMBDA_ARN = module.state_end_lease_lambda.arn
  })

  tags = var.global_tags
}

module "state_get_lease_lambda" {
  source          = "./lambda"
  name            = "state_get_lease-${var.namespace}"
  namespace       = var.namespace
  description     = "Gets the lease for state machines so we have the most up to date lease record"
  global_tags     = var.global_tags
  handler         = "state_get_lease"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    LEASE_DB           = aws_dynamodb_table.leases.id
    PRINCIPAL_DB       = aws_dynamodb_table.principal.id
  }
}

module "state_get_lease_usage_lambda" {
  source          = "./lambda"
  name            = "state_get_lease_usage-${var.namespace}"
  namespace       = var.namespace
  description     = "Gets the lease usage information and upates it"
  global_tags     = var.global_tags
  handler         = "state_get_lease_usage"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    USAGE_TTL          = var.usage_ttl
    ACCOUNT_DB         = aws_dynamodb_table.accounts.id
    PRINCIPAL_DB       = aws_dynamodb_table.principal.id
  }
}

module "state_end_lease_lambda" {
  source          = "./lambda"
  name            = "state_end_lease-${var.namespace}"
  namespace       = var.namespace
  description     = "Ends the lease"
  global_tags     = var.global_tags
  handler         = "state_end_lease"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    USAGE_TTL          = var.usage_ttl
    ACCOUNT_DB         = aws_dynamodb_table.accounts.id
    PRINCIPAL_DB       = aws_dynamodb_table.principal.id
  }
}
