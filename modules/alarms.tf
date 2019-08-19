# Alarms for monitoring across infrastructure on AWS
# Lambda Alarms
# Leases Lambda
resource "aws_cloudwatch_metric_alarm" "lambda-alarm-leases-errors" {
  alarm_name                = "lambda-alarm-leases-errors-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "Errors"
  namespace                 = "AWS/Lambda"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.leases.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-leases-duration" {
  alarm_name          = "lambda-alarm-leases-duration-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  period              = 60
  statistic           = "Maximum"
  threshold           = 15000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.leases.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-leases-throttles" {
  alarm_name          = "lambda-alarm-leases-throttles-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.leases.function_name
  }
}

# ResetSQS Lambda
resource "aws_cloudwatch_metric_alarm" "lambda-alarm-resetsqs-errors" {
  alarm_name                = "lambda-alarm-resetsqs-errors-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "Errors"
  namespace                 = "AWS/Lambda"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.populate_reset_queue.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-resetsqs-duration" {
  alarm_name          = "lambda-alarm-resetsqs-duration-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  period              = 60
  statistic           = "Maximum"
  threshold           = 15000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.populate_reset_queue.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-resetsqs-throttles" {
  alarm_name          = "lambda-alarm-resetsqs-throttles-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.populate_reset_queue.function_name
  }
}

# ExecuteReset Lambda
resource "aws_cloudwatch_metric_alarm" "lambda-alarm-executereset-errors" {
  alarm_name                = "lambda-alarm-exectuereset-errors-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "Errors"
  namespace                 = "AWS/Lambda"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.process_reset_queue.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-executereset-duration" {
  alarm_name          = "lambda-alarm-exectuereset-duration-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  period              = 60
  statistic           = "Maximum"
  threshold           = 15000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.process_reset_queue.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-executereset-throttles" {
  alarm_name          = "lambda-alarm-exectuereset-throttles-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.process_reset_queue.function_name
  }
}

# DynamoDB
resource "aws_cloudwatch_metric_alarm" "dynamodb-account-systemerrors-alarm" {
  alarm_name                = "dynamodb-account-systemerrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "SystemErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_account.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-lease-systemerrors-alarm" {
  alarm_name                = "dynamodb-lease-systemerrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "SystemErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_lease.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-account-usererrors-alarm" {
  alarm_name                = "dynamodb-account-usererrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "UserErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_account.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-lease-usererrors-alarm" {
  alarm_name                = "dynamodb-lease-usererrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "UserErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_lease.name
  }
}

# API Gateway 4xx errors
resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-4xx" {
  alarm_name                = "apigateway-alarm-4xx-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "4XXError"
  namespace                 = "AWS/ApiGateway"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 50
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

# API Gateway 5xx errors
resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-5xx" {
  alarm_name                = "apigateway-alarm-5xx-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "5XXError"
  namespace                 = "AWS/ApiGateway"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-latency" {
  alarm_name          = "apigateway-alarm-latency-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/ApiGateway"
  metric_name         = "Latency"
  period              = 60
  statistic           = "Sum"
  threshold           = 10000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-integ-latency" {
  alarm_name          = "apigateway-alarm-integ-latency-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/ApiGateway"
  metric_name         = "IntegrationLatency"
  period              = 60
  statistic           = "Sum"
  threshold           = 10000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

# Simple Email Service Alarms
resource "aws_cloudwatch_metric_alarm" "ses-bounce-delivery" {
  alarm_name          = "ses-bounced-delivery-${var.namespace}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "Reputation.BounceRate"
  namespace           = "AWS/SES"
  period              = "120"
  statistic           = "Maximum"
  threshold           = "1"
  alarm_description   = "This metric monitors email bounce rate"
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]
}

resource "aws_cloudwatch_metric_alarm" "ses-complaint-delivery" {
  alarm_name          = "ses-complaint-delivery-${var.namespace}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "Reputation.ComplaintRate"
  namespace           = "AWS/SES"
  period              = "120"
  statistic           = "Maximum"
  threshold           = "1"
  alarm_description   = "This metric monitors email complaint rate"
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]
}

resource "aws_cloudwatch_metric_alarm" "ses-reject-delivery" {
  alarm_name          = "ses-reject-delivery-${var.namespace}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "Reputation.RejectRate"
  namespace           = "AWS/SES"
  period              = "120"
  statistic           = "Maximum"
  threshold           = "1"
  alarm_description   = "This metric monitors email reject rate"
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]
}
