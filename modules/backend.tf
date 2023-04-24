
terraform {
  backend "s3" {
    bucket = "local-deployment-dce-tfstate"
    region = "eu-west-1"
    key    = "dce.tfstate"
  }
}

