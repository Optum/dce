module "publish_locks_lambda" {
  source          = "./lambda"
  name            = "publish_locks-${var.namespace}"
  namespace       = var.namespace
  description     = "Publishes to SNS lease-locked and lease-unlocked topics in response to DB changes"
  global_tags     = var.global_tags
  handler         = "publish_locks"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    AWS_CURRENT_REGION       = var.aws_region
    ACCOUNT_DB               = aws_dynamodb_table.redbox_account.id
    LEASE_DB                 = aws_dynamodb_table.redbox_lease.id
    LEASE_LOCKED_TOPIC_ARN   = aws_sns_topic.lease_locked.arn
    LEASE_UNLOCKED_TOPIC_ARN = aws_sns_topic.lease_unlocked.arn
  }
}

resource "aws_lambda_event_source_mapping" "publish_locks_from_dynamo_db" {
  event_source_arn  = aws_dynamodb_table.redbox_lease.stream_arn
  function_name     = module.publish_locks_lambda.name
  batch_size        = 1
  starting_position = "LATEST"
}

resource "aws_iam_role_policy" "publish_locks_lambda_dynamo_db" {
  role   = module.publish_locks_lambda.execution_role_name
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
        "Resource": "${aws_dynamodb_table.redbox_lease.stream_arn}"
    },
    {
        "Effect": "Allow",
        "Action": [
            "sns:Publish"
        ],
        "Resource": [
            "*"
        ]
    }
  ]
}
POLICY
}
