data "aws_route53_zone" "observe_blunderdome" {
  name         = "${var.custom_zone_name}."
  private_zone = false
}

resource "aws_route53_record" "custom" {
  zone_id = aws_route53_zone.observe_blunderdome.zone_id
  name    = "${var.custom_record_name}.${var.custom_zone_name}"
  type    = "A"

  alias {
    name                   = 
    zone_id                = aws_elb.main.zone_id
    evaluate_target_health = true
  }
}

data "aws_acm_certificate" "custom" {
  domain   = "${var.custom_record_name}.${var.custom_zone_name}"
  statuses = ["ISSUED"]
}