module "publish_lease_events_lambda" {
  source          = "./lambda"
  name            = "publish_lease_events-${var.namespace}"
  namespace       = var.namespace
  description     = "Publishes lease change events to SNS and SQS in response to DB changes"
  global_tags     = var.global_tags
  handler         = "publish_lease_events"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    AWS_CURRENT_REGION       = var.aws_region
    ACCOUNT_DB               = aws_dynamodb_table.accounts.id
    LEASE_DB                 = aws_dynamodb_table.leases.id
    LEASE_LOCKED_TOPIC_ARN   = aws_sns_topic.lease_locked.arn
    LEASE_UNLOCKED_TOPIC_ARN = aws_sns_topic.lease_unlocked.arn
    RESET_QUEUE_URL          = aws_sqs_queue.account_reset.id
  }
}

resource "aws_lambda_event_source_mapping" "publish_lease_events_from_dynamo_db" {
  event_source_arn  = aws_dynamodb_table.leases.stream_arn
  function_name     = module.publish_lease_events_lambda.name
  batch_size        = 1
  starting_position = "LATEST"
}

resource "aws_iam_role_policy" "publish_lease_events_lambda_dynamo_db" {
  role   = module.publish_lease_events_lambda.execution_role_name
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
        "Effect": "Allow",
        "Action": [
            "dynamodb:DescribeStream",
            "dynamodb:GetRecords",
            "dynamodb:GetShardIterator",
            "dynamodb:ListStreams"
        ],
        "Resource": "${aws_dynamodb_table.leases.stream_arn}"
    },
    {
        "Effect": "Allow",
        "Action": [
            "sns:Publish"
        ],
        "Resource": [
            "*"
        ]
    },
    {
        "Effect": "Allow",
        "Action": [
            "sqs:Publish"
        ],
        "Resource": [
            "*"
        ]
    }
  ]
}
POLICY
}
