resource "aws_iam_role" "redbox_lambda_execution" {
  name_prefix = "redbox-lambda-${var.namespace}"

  assume_role_policy = <<JSON
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowLambda",
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": ["lambda.amazonaws.com", "apigateway.amazonaws.com"]
      },
      "Effect": "Allow"
    }
  ]
}
JSON

  tags = var.global_tags
}

# Allow Lambdas to write logs, etc.
resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Allow Lambdas to work with SSM
resource "aws_iam_role_policy_attachment" "lambda_ssm" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMFullAccess"
}


# Allow Lambdas to work with DynamoDD
resource "aws_iam_role_policy_attachment" "lambda_dynamodb" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
}

# Allow Lambdas to work with SQS
resource "aws_iam_role_policy_attachment" "lambda_sqs" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSQSFullAccess"
}

# Allow Lambdas to execute CodeBuild
resource "aws_iam_role_policy_attachment" "lambda_codebuild" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/AWSCodeBuildDeveloperAccess"
}


# Allow Lambdas to work with SNS
resource "aws_iam_role_policy_attachment" "lambda_sns" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSNSFullAccess"
}

# Allow Lambdas to work with S3
resource "aws_iam_role_policy_attachment" "lambda_s3" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}

# Allow cloudwatch logs for API Gateway
resource "aws_iam_role_policy_attachment" "gateway_logs" {
  role       = aws_iam_role.redbox_lambda_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
}

# Allow Lambda to assume roles
resource "aws_iam_role_policy" "lambda_assume_role" {
  role   = aws_iam_role.redbox_lambda_execution.name
  policy = <<JSON
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "sts:AssumeRole",
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
JSON
}