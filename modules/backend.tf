
terraform {
  backend "s3" {
    bucket = "local-dce-tfstate"
    region = "us-east-1"
    key    = "dce.tfstate"
  }
}

