terraform {
  required_version = ">= 1.0.3"

  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "delegator"

    workspaces {
      prefix = "countdown-"
    }
  }

  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region  = "ap-southeast-2"
  profile = "countdown"
}

provider "aws" {
  alias   = "north_virginia"
  region  = "us-east-1"
  profile = "countdown"
}
