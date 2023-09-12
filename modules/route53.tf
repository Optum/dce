resource "aws_route53_zone" "main" {
  name = "observe-blunderdome.com"
}

resource "aws_route53_zone" "dce" {
  name = "dce.observe-blunderdome.com"
}

resource "aws_route53_record" "dce" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "dce.observe-blunderdome.com"
  type    = "CNAME"
  ttl     = "30"
  records = [
    aws_api_gateway_stage.api.invoke_url
  ]
}