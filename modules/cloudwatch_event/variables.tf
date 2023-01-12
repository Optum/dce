variable "lambda_function_arn" {
  type = string
}
variable "schedule_expression" {
  type = string
}
variable "name" {
  type = string
}
variable "description" {
  type = string
}

variable "enabled" {
  type    = bool
  default = true
}
