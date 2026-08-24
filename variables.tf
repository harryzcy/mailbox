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

variable "aws_s3_bucket_override" {
  description = "Email bucket name. Required for now: bucket names are global, so no default reliably works."
  type        = string
  sensitive   = true
}

variable "aws_s3_artifacts_bucket_override" {
  description = "Build artifacts bucket, where packages are signed. Required for now: bucket names are global, so no default reliably works."
  type        = string
  sensitive   = true
}

variable "ses_receipt_rule_set_name" {
  description = "Existing SES receipt rule set to manage a rule in. Leave empty to skip SES entirely."
  type        = string
  default     = ""
}

variable "ses_receipt_rule_name" {
  description = "Existing SES receipt rule to point at email_receive. Leave empty to skip SES entirely."
  type        = string
  default     = ""
}

variable "aws_dynamodb_table_override" {
  description = "Override for the DynamoDB table name (optional)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "aws_dynamodb_point_in_time_recovery" {
  description = "Continuous backups on the managed DynamoDB table. Incurs additional cost."
  type        = bool
  default     = true
}

variable "github_repository" {
  description = "owner/repo allowed to assume the CI role. Leave empty to create no OIDC resources."
  type        = string
  default     = ""
}

variable "github_oidc_provider_arn" {
  description = "Existing account-global GitHub OIDC provider ARN. Required to enable the CI role."
  type        = string
  default     = ""
}

variable "tf_state_bucket" {
  description = "State bucket name, so the CI role can be scoped to it"
  type        = string
  sensitive   = true
  default     = ""
}

locals {
  project_name_env             = "${var.project_name}-${var.environment}"
  aws_dynamodb_table_name      = var.aws_dynamodb_table_override != "" ? var.aws_dynamodb_table_override : "${var.project_name}-${var.environment}"
  aws_dynamodb_original_index  = "OriginalMessageIDIndex"
  aws_dynamodb_time_index      = "TimeIndex"
  aws_s3_emails_bucket_name    = var.aws_s3_bucket_override
  aws_s3_artifacts_bucket_name = var.aws_s3_artifacts_bucket_override
  aws_sqs_queue_name           = "${var.project_name}-${var.environment}"
  webhook_url                  = ""

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
      function   = "emails_untrash"
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
      function   = "threads_untrash"
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

  # several functions serve more than one route
  lambda_packages = toset(concat(
    [for f in local.lambda_functions : f.function],
    ["email_receive"],
  ))
}
