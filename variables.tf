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

  lambda_functions = {
    emails_list = {
      function   = "emails_list"
      httpMethod = "GET"
      httpPath   = "/emails"
      arnPath    = "/emails"
    },
    emails_get = {
      function   = "emails_get"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}"
      arnPath    = "/emails/*"
    },
    emails_getRaw = {
      function   = "emails_getRaw"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}/raw"
      arnPath    = "/emails/*/raw"
    },
    emails_getRawDownload = {
      function   = "emails_getRaw"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}/download"
      arnPath    = "/emails/*/download"
    },
    emails_getContentAttachments = {
      function   = "emails_getContent"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}/attachments/{contentID}"
      arnPath    = "/emails/*/attachments/*"
    },
    emails_getContentInlines = {
      function   = "emails_getContent"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}/inlines/{contentID}"
      arnPath    = "/emails/*/inlines/*"
    },
    emails_getContentOthers = {
      function   = "emails_getContent"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}/others/{contentID}"
      arnPath    = "/emails/*/others/*"
    },
    emails_read = {
      function   = "emails_read"
      httpMethod = "POST"
      httpPath   = "/emails/{messageID}/read"
      arnPath    = "/emails/*/read"
    },
    emails_unread = {
      function   = "emails_read"
      httpMethod = "POST"
      httpPath   = "/emails/{messageID}/unread"
      arnPath    = "/emails/*/unread"
    },
    emails_trash = {
      function   = "emails_trash"
      httpMethod = "POST"
      httpPath   = "/emails/{messageID}/trash"
      arnPath    = "/emails/*/trash"
    },
    emails_untrash = {
      function   = "emails_trash"
      httpMethod = "POST"
      httpPath   = "/emails/{messageID}/untrash"
      arnPath    = "/emails/*/untrash"
    },
    emails_delete = {
      function   = "emails_delete"
      httpMethod = "DELETE"
      httpPath   = "/emails/{messageID}"
      arnPath    = "/emails/*"
    },
    emails_create = {
      function   = "emails_create"
      httpMethod = "POST"
      httpPath   = "/emails"
      arnPath    = "/emails"
    },
    emails_save = {
      function   = "emails_save"
      httpMethod = "PUT"
      httpPath   = "/emails/{messageID}"
      arnPath    = "/emails/*"
    },
    emails_send = {
      function   = "emails_send"
      httpMethod = "POST"
      httpPath   = "/emails/{messageID}/send"
      arnPath    = "/emails/*/send"
    },
    emails_reparse = {
      function   = "emails_reparse"
      httpMethod = "POST"
      httpPath   = "/emails/{messageID}/reparse"
      arnPath    = "/emails/*/reparse"
    },
    threads_get = {
      function   = "threads_get"
      httpMethod = "GET"
      httpPath   = "/threads/{threadID}"
      arnPath    = "/threads/*"
    },
    threads_delete = {
      function   = "threads_delete"
      httpMethod = "DELETE"
      httpPath   = "/threads/{threadID}"
      arnPath    = "/threads/*"
    },
    threads_trash = {
      function   = "threads_trash"
      httpMethod = "POST"
      httpPath   = "/threads/{threadID}/trash"
      arnPath    = "/threads/*/trash"
    },
    threads_untrash = {
      function   = "threads_trash"
      httpMethod = "POST"
      httpPath   = "/threads/{threadID}/untrash"
      arnPath    = "/threads/*/untrash"
    },
    info = {
      function   = "info"
      httpMethod = "GET"
      httpPath   = "/info"
      arnPath    = "/info"
    }
  }
}
