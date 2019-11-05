variable "namespace" {
  type = string
}
variable "environment" {
  type    = map(string)
  default = { TERRAFORM = "true" }
}
variable "global_tags" {
  type = map(string)
}
variable "name" {
  type = string
}
variable "description" {
  type = string
}
variable "handler" {
  type = string
}
variable "alarm_topic_arn" {
  type        = string
  description = "ARN of SNS Topic, for alarm notifications"
}

variable "namespace_prefix" {
  type = string
}
