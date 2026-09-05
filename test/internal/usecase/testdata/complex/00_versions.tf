terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    xaligo = {
      source  = "xaligo/xaligo"
      version = "0.1.0"
    }
  }
}

provider "xaligo" {
  export = "enable"
}

provider "aws" {
  region = var.region
}

variable "region" {
  type    = string
  default = "ap-northeast-1"
}

variable "ingress_rules" {
  type = map(object({
    port = number
    cidr = string
  }))
  default = {
    https = {
      port = 443
      cidr = "0.0.0.0/0"
    }
  }
}

variable "database_password" {
  type      = string
  sensitive = true
}

locals {
  environment = "complex-test"
  web_nodes = {
    primary   = "t3.micro"
    secondary = "t3.small"
  }
}
