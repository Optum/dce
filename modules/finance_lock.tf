# SQS Queue, for triggering finance lock on account
resource "aws_sqs_queue" "finance_lock_queue" {
  name                       = "redbox-finance-lock-${var.namespace}"
  tags                       = var.global_tags
  visibility_timeout_seconds = 6400
}

# SQS Queue Policy to allow Redbox Accounts to send messages to it 
resource "aws_sqs_queue_policy" "finance_lock_queue_policy" {
  queue_url = aws_sqs_queue.finance_lock_queue.id

  policy = <<JSON
{
  "Version": "2012-10-17",
  "Id": "FinanceLockQueuePolicy",
  "Statement": [
    {
      "Sid": "SendMessage",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "sqs:SendMessage",
      "Resource": "${aws_sqs_queue.finance_lock_queue.arn}",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": ["${var.organization_id}"]
        }
      }
    }
  ]
}
JSON

}

# Finance Lock lambda function
resource "aws_lambda_function" "financelock" {
  function_name = "financelock-${var.namespace}"
  description = "Reads SQS, updates DB, and sends to SQS for Finance Lock"
  runtime = "go1.x"
  handler = "financelock"
  role = aws_iam_role.redbox_lambda_execution.arn
  timeout = 300

  environment {
    variables = {
      DEBUG = "false"
      NAMESPACE = var.namespace
      RESET_SQS_URL = aws_sqs_queue.account_reset.id
      ACCOUNT_DB = aws_dynamodb_table.redbox_account.id
      ASSIGNMENT_DB = aws_dynamodb_table.redbox_account_assignment.id
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

# Event Source Map for SQS queue to trigger lambda
resource "aws_lambda_event_source_mapping" "event_source_mapping" {
  batch_size = 1
  event_source_arn = aws_sqs_queue.finance_lock_queue.arn
  enabled = true
  function_name = aws_lambda_function.financelock.id
}

