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
  region  = "ap-southeast-2"
  profile = "countdown"
}
