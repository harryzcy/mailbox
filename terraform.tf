terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.46.0"
    }
  }

  required_version = ">= 1.14"
}
