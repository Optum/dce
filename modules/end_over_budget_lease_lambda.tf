module "end_over_budget_lease_lambda" {
  source          = "./lambda"
  name            = "end_over_budget_lease-${var.namespace}"
  namespace       = var.namespace
  description     = "Ends over budget leases in response to dynamodb stream events"
  global_tags     = var.global_tags
  handler         = "end_over_budget_lease"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    AWS_CURRENT_REGION       = var.aws_region
    PRINCIPAL_DB                 = aws_dynamodb_table.principal.id
  }
}

resource "aws_lambda_event_source_mapping" "principal_events_from_dynamo_db" {
  event_source_arn  = aws_dynamodb_table.principal.stream_arn
  function_name     = module.end_over_budget_lease_lambda.name
  batch_size        = 1
  starting_position = "LATEST"
}

resource "aws_iam_role_policy" "end_over_budget_lease_lambda_dynamo_db" {
  role   = module.end_over_budget_lease_lambda.execution_role_name
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
        "Resource": "${aws_dynamodb_table.principal.stream_arn}"
    }
  ]
}
POLICY
}
