variable "aws_region" {
  description = "The AWS region to deploy resources in"
  type        = string
  default     = "us-west-2"
}

provider "aws" {
  region = var.aws_region
}
