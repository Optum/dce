# Creates SES email entry
resource "aws_ses_email_identity" "dcs_from_email_address" {
  email = var.budget_notification_from_email
}

# Send delivery failures and bounces to SNS topic
resource "aws_ses_configuration_set" "dcs_ses" {
  name = "dcs_ses_${var.namespace}"
}

resource "aws_ses_event_destination" "dcs_ses_cloudwatch" {
  name                   = "event-destination-cloudwatch-${var.namespace}"
  configuration_set_name = "${aws_ses_configuration_set.dcs_ses.name}"
  enabled                = true

  matching_types = [
    "bounce",
    "reject",
    "complaint",
    "renderingFailure",
  ]

  cloudwatch_destination {
    default_value  = "Failed_Send"
    dimension_name = "RB_SES_FAILURE"
    value_source   = "emailHeader"
  }
}
