module "cloudwatch_log-group" {
  source   = "terraform-aws-modules/cloudwatch/aws//modules/log-group"
  version  = "=3.3.0"
  for_each = local.cloudwatch_log_groups_set

  name              = each.value
  create            = false
  retention_in_days = 7
}