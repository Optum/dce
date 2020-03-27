resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  alarm_name                = "${var.name}-errors"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "Errors"
  namespace                 = "AWS/Lambda"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 2
  insufficient_data_actions = []
  alarm_actions             = [var.alarm_topic_arn]

  dimensions = {
    FunctionName = aws_lambda_function.fn.function_name
  }

  tags = var.global_tags
}

resource "aws_cloudwatch_metric_alarm" "lambda_duration" {
  alarm_name          = "${var.name}-duration"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  period              = 60
  statistic           = "Maximum"
  threshold           = 15000
  alarm_actions       = [var.alarm_topic_arn]

  dimensions = {
    FunctionName = aws_lambda_function.fn.function_name
  }

  tags = var.global_tags
}

resource "aws_cloudwatch_metric_alarm" "lambda_throttles" {
  alarm_name          = "${var.name}-throttles"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_actions       = [var.alarm_topic_arn]

  dimensions = {
    FunctionName = aws_lambda_function.fn.function_name
  }

  tags = var.global_tags
}

resource "aws_sqs_queue" "lambda_dlq" {
  name                       = "${var.name}-dlq"
  tags                       = var.global_tags
  visibility_timeout_seconds = 30
}

resource "aws_cloudwatch_metric_alarm" "dlq_not_empty" {
  alarm_name          = "${var.name}-dlq-not-empty"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 60
  threshold           = 0
  statistic           = "Sum"
  alarm_actions       = [var.alarm_topic_arn]

  dimensions {
    QueueName = aws_sqs_queue.lambda_dlq.arn
  }

  tags = var.global_tags
}
