# Alarms for monitoring across infrastructure on AWS
# Lambda Alarms
# Acctmgr Lambda
resource "aws_cloudwatch_metric_alarm" "lambda-alarm-acctmgr-errors" {
  alarm_name                = "lambda-alarm-acctmgr-errors-${var.namespace}"
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
    FunctionName = aws_lambda_function.acctmgr.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-acctmgr-duration" {
  alarm_name          = "lambda-alarm-acctmgr-duration-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  period              = 60
  statistic           = "Maximum"
  threshold           = 15000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.acctmgr.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-acctmgr-throttles" {
  alarm_name          = "lambda-alarm-acctmgr-throttles-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  period              = 60
  statistic           = "Maximum"
  threshold           = 1
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.acctmgr.function_name
  }
}

# FinanceLock Lambda
resource "aws_cloudwatch_metric_alarm" "lambda-alarm-financelock-errors" {
  alarm_name                = "lambda-alarm-financelock-errors-${var.namespace}"
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
    FunctionName = aws_lambda_function.financelock.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-financelock-duration" {
  alarm_name          = "lambda-alarm-financelock-duration-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  period              = 60
  statistic           = "Maximum"
  threshold           = 15000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.financelock.function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "lambda-alarm-financelock-throttles" {
  alarm_name          = "lambda-alarm-financelock-throttles-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  period              = 60
  statistic           = "Maximum"
  threshold           = 1
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.financelock.function_name
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
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.global_reset_enqueue.function_name
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
    FunctionName = aws_lambda_function.global_reset_enqueue.function_name
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
  threshold           = 1
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.global_reset_enqueue.function_name
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
    FunctionName = aws_lambda_function.execute_reset.function_name
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
    FunctionName = aws_lambda_function.execute_reset.function_name
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
  threshold           = 1
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    FunctionName = aws_lambda_function.execute_reset.function_name
  }
}

# SQS Alarms
# Account Reset Failed SQS
resource "aws_cloudwatch_metric_alarm" "sqs-acctreset-alarm" {
  alarm_name                = "sqs-acctreset-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "NumberOfNotificationsFailed"
  namespace                 = "AWS/SQS"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    QueueName = aws_sqs_queue.account_reset.name
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
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_account.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-assignment-systemerrors-alarm" {
  alarm_name                = "dynamodb-assignment-systemerrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "SystemErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_account_assignment.name
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
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_account.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-assignment-usererrors-alarm" {
  alarm_name                = "dynamodb-assignment-usererrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "UserErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 1
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.redbox_account_assignment.name
  }
}

# API Gateway
resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-5xx" {
  alarm_name                = "apigateway-alarm-5xx-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "5xxError"
  namespace                 = "AWS/ApiGateway"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 1
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
  threshold           = 1000
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
  threshold           = 500
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

