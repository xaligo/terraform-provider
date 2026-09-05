terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }

    xaligo = {
      source = "xaligo/xaligo"
    }
  }
}

variable "ministack_endpoint" {
  description = "MiniStack endpoint used by the local AWS provider"
  type        = string
  default     = "http://127.0.0.1:4566"
}

provider "aws" {
  region                      = "ap-northeast-1"
  access_key                  = "000000000000"
  secret_key                  = "test"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true

  endpoints {
    ec2 = var.ministack_endpoint
  }
}

provider "xaligo" {
  export = "enable"
}

module "infrastructure" {
  source = "./source"
}

resource "xaligo_diagram" "architecture" {
  source_dir  = abspath("${path.module}/source")
  output_path = abspath("${path.module}/generated.xal")
  frame_id    = "main"
  title       = "Simple VPC"
}
