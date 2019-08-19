locals {
  redbox_principal_role_name   = "RedboxPrincipal${var.namespace == "prod" ? "" : "-${var.namespace}"}"
  redbox_principal_policy_name = "RedboxPrincipalDefaultPolicy${var.namespace == "prod" ? "" : "-${var.namespace}"}"
}

resource "aws_lambda_function" "accounts_lambda" {
  function_name = "accounts-${var.namespace}"
  description   = "Handles API requests to the /accounts endpoint"
  runtime       = "go1.x"
  handler       = "accounts"
  role          = aws_iam_role.redbox_lambda_execution.arn
  timeout       = 300

  environment {
    variables = {
      DEBUG                          = "false"
      NAMESPACE                      = var.namespace
      AWS_CURRENT_REGION             = var.aws_region
      ACCOUNT_DB                     = aws_dynamodb_table.redbox_account.id
      LEASE_DB                       = aws_dynamodb_table.redbox_lease.id
      RESET_SQS_URL                  = aws_sqs_queue.account_reset.id
      ACCOUNT_CREATED_TOPIC_ARN      = aws_sns_topic.account_created.arn
      ACCOUNT_DELETED_TOPIC_ARN      = aws_sns_topic.account_deleted.arn
      PRINCIPAL_ROLE_NAME            = local.redbox_principal_role_name
      PRINCIPAL_POLICY_NAME          = local.redbox_principal_policy_name
      PRINCIPAL_IAM_DENY_TAGS        = join(",", var.principal_iam_deny_tags)
      PRINCIPAL_MAX_SESSION_DURATION = 14400
      TAG_ENVIRONMENT                = var.namespace == "prod" ? "PROD" : "NON-PROD"
      TAG_CONTACT                    = lookup(var.global_tags, "Contact")
      TAG_APP_NAME                   = lookup(var.global_tags, "AppName")
    }
  }

  # Stub an application deployment
  # (deployments will be managed outside terraform)
  filename = data.archive_file.lambda_code_stub.output_path

  lifecycle {
    # Filename will change, as new application deployments are pushed.
    # Prevent terraform from reverting to old application deployments
    # We're not using terraform to manage application deployments
    ignore_changes = [filename]
  }

  tags = var.global_tags
}

resource "aws_sns_topic" "account_created" {
  name = "redbox-account-created-${var.namespace}"
  tags = var.global_tags
}

resource "aws_sns_topic" "account_deleted" {
  name = "redbox-account-deleted-${var.namespace}"
  tags = var.global_tags
}
