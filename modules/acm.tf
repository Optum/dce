data "aws_route53_zone" "observe_blunderdome" {
  name         = "${var.custom_zone_name}."
  private_zone = false
}

data "aws_api_gateway_domain_name" "custom" {
  domain_name = aws_api_gateway_rest_api.gateway_api
}

resource "aws_route53_record" "custom" {
  zone_id = data.aws_route53_zone.observe_blunderdome.zone_id
  name    = "${var.custom_record_name}.${var.custom_zone_name}"
  type    = "A"

  alias {
    zone_id                = data.aws_api_gateway_domain_name.regional_zone_id
    evaluate_target_health = true
  }
}

data "aws_acm_certificate" "custom" {
  domain   = "${var.custom_record_name}.${var.custom_zone_name}"
  statuses = ["ISSUED"]
}