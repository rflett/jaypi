terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "delegator"

    workspaces {
      prefix = "countdown-"
    }
  }
}

provider "aws" {
  version = "~> 2.0"
  region  = "ap-southeast-2"
  profile = "countdown"
}
