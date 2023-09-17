data "aws_route53_zone" "observe_blunderdome" {
  name         = "${var.custom_zone_name}."
  private_zone = false
}

resource "aws_route53_record" "custom" {
  zone_id = data.aws_route53_zone.observe_blunderdome.zone_id
  name    = "${var.custom_record_name}.${var.custom_zone_name}"
  type    = "A"

  alias {
    name                   = aws_api_gateway_domain_name.gateway_api.regional_domain_name
    zone_id                = aws_api_gateway_domain_name.gateway_api.regional_zone_id
    evaluate_target_health = true
  }
}

data "aws_acm_certificate" "custom" {
  domain   = "${var.custom_record_name}.${var.custom_zone_name}"
  statuses = ["ISSUED"]
}