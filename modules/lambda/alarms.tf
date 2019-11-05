resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  alarm_name                = "${var.namespace_prefix}-${var.name}-errors-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "Errors"
  namespace                 = "AWS/Lambda"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [var.alarm_topic_arn]

  dimensions = {
    FunctionName = aws_lambda_function.fn.function_name
  }

  tags = var.global_tags
}

resource "aws_cloudwatch_metric_alarm" "lambda_duration" {
  alarm_name          = "${var.namespace_prefix}-${var.name}-duration-${var.namespace}"
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
  alarm_name          = "${var.namespace_prefix}-${var.name}-throttles-${var.namespace}"
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
