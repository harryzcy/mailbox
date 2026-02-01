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

variable "aws_dynamodb_table_name" {
  description = "The name of the DynamoDB table for mailbox data"
  type        = string
  default     = "${var.project_name}-${var.environment}"
}

variable "aws_dynamodb_time_index" {
  description = "The name of the DynamoDB time index"
  type        = string
  default     = "TimeIndex"
}

variable "aws_s3_bucket_name" {
  description = "The name of the S3 bucket for mailbox attachments"
  type        = string
  default     = "${var.project_name}-${var.environment}"
}

variable "aws_sqs_queue_name" {
  description = "The name of the SQS queue for mailbox messages"
  type        = string
  default     = "${var.project_name}-${var.environment}"
}
