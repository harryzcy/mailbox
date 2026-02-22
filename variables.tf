variable "project_name" {
  description = "The name of the project"
  type        = string
  default     = "mailbox-v2"
}

variable "environment" {
  description = "Deployment environment (e.g., dev, prod)"
  type        = string
  default     = "dev"
}

variable "aws_region" {
  description = "The AWS region to deploy resources in"
  type        = string
  default     = "us-west-2"
}

locals {
  project_name_env            = "${var.project_name}-${var.environment}"
  aws_dynamodb_table_name     = "${var.project_name}-${var.environment}"
  aws_dynamodb_original_index = "OriginalMessageIDIndex"
  aws_dynamodb_time_index     = "TimeIndex"
  aws_s3_bucket_name          = "${var.project_name}-${var.environment}"
  aws_sqs_queue_name          = "${var.project_name}-${var.environment}"
  webhook_url                 = ""
}
