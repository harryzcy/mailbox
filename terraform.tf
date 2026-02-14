terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.32.0"
    }
  }

  required_version = ">= 1.14"
}
