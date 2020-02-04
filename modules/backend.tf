terraform {
  backend "s3" {
    bucket = "391501768339-redbox-tfstate"
    key    = "github-pr-2507/terraform.tfstate"
    region = "us-east-1"
  }
}