# Make a topic
resource "aws_sns_topic" "alarms_topic" {
  name = "alarms-${var.namespace}"
}

resource "aws_sns_topic_policy" "alarms_policy" {
  arn    = aws_sns_topic.alarms_topic.arn
  policy = <<POLICY
  { 
    "Version": "2012-10-17",
    "Id": "${var.namespace}-alarm_policy_ID",
    "Statement": [
      {
        "Sid": "alarm_statement_ID",
        "Effect": "Allow",
        "Principal": "*",
        "Action": [
          "SNS:Subscribe",
          "SNS:SetTopicAttributes",
          "SNS:RemovePermission",
          "SNS:Receive",
          "SNS:Publish",
          "SNS:ListSubscriptionsByTopic",
          "SNS:GetTopicAttributes",
          "SNS:DeleteTopic",
          "SNS:AddPermission"
        ],
        "Resource": "${aws_sns_topic.alarms_topic.arn}",
        "Condition": {
          "StringEquals": {
            "AWS:SourceOwner": ["${data.aws_caller_identity.current.account_id}"]
          }
        }
      },
      {
        "Sid": "Allow_CloudwatchEvents",
        "Effect": "Allow",
        "Principal": {
          "AWS":"*"
        },
        "Action": "SNS:Publish",
        "Resource": "${aws_sns_topic.alarms_topic.arn}",
        "Condition": {
          "StringEquals": {
            "AWS:SourceOwner": ["${data.aws_caller_identity.current.account_id}"]
          }
        }
      }
    ]
  }
POLICY
}

resource "aws_sns_topic" "alarms_topic_low_urgency" {
  name = "alarms-${var.namespace}"
}

resource "aws_sns_topic_policy" "alarms_policy_low_urgency" {
  arn    = aws_sns_topic.alarms_topic_low_urgency.arn
  policy = <<POLICY
  {
    "Version": "2012-10-17",
    "Id": "${var.namespace}-alarm_policy_ID",
    "Statement": [
      {
        "Sid": "alarm_statement_ID",
        "Effect": "Allow",
        "Principal": "*",
        "Action": [
          "SNS:Subscribe",
          "SNS:SetTopicAttributes",
          "SNS:RemovePermission",
          "SNS:Receive",
          "SNS:Publish",
          "SNS:ListSubscriptionsByTopic",
          "SNS:GetTopicAttributes",
          "SNS:DeleteTopic",
          "SNS:AddPermission"
        ],
        "Resource": "${aws_sns_topic.alarms_topic_low_urgency.arn}",
        "Condition": {
          "StringEquals": {
            "AWS:SourceOwner": ["${data.aws_caller_identity.current.account_id}"]
          }
        }
      },
      {
        "Sid": "Allow_CloudwatchEvents",
        "Effect": "Allow",
        "Principal": {
          "AWS":"*"
        },
        "Action": "SNS:Publish",
        "Resource": "${aws_sns_topic.alarms_topic_low_urgency.arn}",
        "Condition": {
          "StringEquals": {
            "AWS:SourceOwner": ["${data.aws_caller_identity.current.account_id}"]
          }
        }
      }
    ]
  }
POLICY
}
