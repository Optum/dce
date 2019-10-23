variable "name" {
  type = string
}

variable "namespace" {
  type = string
}

variable "api_gateway_arn" {
  type = string
}

variable "callback_urls" {
  type    = list(string)
  default = ["http://localhost:8080"]
}

variable "logout_urls" {
  type    = list(string)
  default = ["http://localhost:8080"]
}

variable "identity_providers" {
  type    = list(string)
  default = ["COGNITO"]
}
