provider "aws" {
  region = var.aws_region
}

resource "aws_apigatewayv2_api" "mailbox_api" {
  name          = "${var.project_name}-api"
  protocol_type = "HTTP"
}

resource "aws_cloudwatch_log_group" "mailbox_api_access_logs" {
  name              = "/aws/apigateway/${var.project_name}-api-access-logs"
  retention_in_days = 365
}

resource "aws_apigatewayv2_stage" "mailbox_api_default" {
  api_id      = aws_apigatewayv2_api.mailbox_api.id
  name        = "$default"
  auto_deploy = true

  default_route_settings {
    throttling_burst_limit = 100
    throttling_rate_limit  = 50
  }

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.mailbox_api_access_logs.arn
    format = jsonencode({
      requestId        = "$context.requestId"
      ip               = "$context.identity.sourceIp"
      requestTime      = "$context.requestTime"
      httpMethod       = "$context.httpMethod"
      path             = "$context.path"
      status           = "$context.status"
      protocol         = "$context.protocol"
      responseLength   = "$context.responseLength"
      integrationError = "$context.integrationErrorMessage"
    })
  }
}

resource "aws_iam_role" "lambda_exec_role" {
  name = "${local.project_name_env}-lambda-exec-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Action = "sts:AssumeRole",
      Effect = "Allow",
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

data "aws_caller_identity" "current" {}

resource "aws_iam_policy" "lambda_dynamodb_s3" {
  name        = "${local.project_name_env}-lambda-dynamodb-s3-policy"
  description = "IAM policy granting Lambda functions access to DynamoDB, S3, SQS, and SES resources"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:DeleteItem",
          "dynamodb:UpdateItem",
          "dynamodb:BatchGetItem",
          "dynamodb:BatchWriteItem"
        ]
        Resource = [
          "arn:aws:dynamodb:${var.aws_region}:${data.aws_caller_identity.current.account_id}:table/${local.aws_dynamodb_table_name}",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          "arn:aws:dynamodb:${var.aws_region}:${data.aws_caller_identity.current.account_id}:table/${local.aws_dynamodb_table_name}/index/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:DeleteObject"
        ]
        Resource = "arn:aws:s3:::${local.aws_s3_bucket_name}/*"
      },
      {
        Effect   = "Allow"
        Action   = "s3:ListBucket"
        Resource = "arn:aws:s3:::${local.aws_s3_bucket_name}"
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:GetQueueUrl",
          "sqs:SendMessage"
        ]
        Resource = "arn:aws:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${local.aws_sqs_queue_name}"
      },
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "ses:SendRawEmail"
        ]
        Resource = "arn:aws:ses:${var.aws_region}:${data.aws_caller_identity.current.account_id}:identity/*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_dynamodb_s3" {
  role       = aws_iam_role.lambda_exec_role.name
  policy_arn = aws_iam_policy.lambda_dynamodb_s3.arn
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_signer_signing_profile" "lambda_signing_profile" {
  platform_id = "AWSLambda-SHA384-ECDSA"
  name_prefix = replace("${local.project_name_env}-lambda", "-", "_")

  signature_validity_period {
    value = 5
    type  = "YEARS"
  }
}

resource "aws_lambda_code_signing_config" "lambda_code_signing" {
  allowed_publishers {
    signing_profile_version_arns = [
      aws_signer_signing_profile.lambda_signing_profile.version_arn
    ]
  }

  policies {
    # TODO: restore "Enforce" once build artifacts are signed with AWS Signer.
    # Signer only signs S3 objects, so that also means publishing the zips to a
    # versioned bucket and deploying via s3_bucket/s3_key instead of filename.
    untrusted_artifact_on_deployment = "Warn"
  }

  description = "Code signing configuration for ${local.project_name_env} Lambda functions"
}

resource "aws_cloudwatch_log_group" "function_logs" {
  for_each          = tomap(local.lambda_functions)
  name              = "/aws/lambda/${local.project_name_env}-${each.key}"
  retention_in_days = 365
}

resource "aws_cloudwatch_log_group" "email_receive_logs" {
  name              = "/aws/lambda/${local.project_name_env}-email_receive"
  retention_in_days = 365
}

resource "aws_lambda_function" "email_receive" {
  #checkov:skip=CKV_AWS_116: TODO: add SQS for DLQ
  function_name                  = "${local.project_name_env}-email_receive"
  filename                       = "bin/email_receive.zip"
  handler                        = "bootstrap"
  runtime                        = "provided.al2023"
  role                           = aws_iam_role.lambda_exec_role.arn
  source_code_hash               = filebase64sha256("bin/email_receive.zip")
  reserved_concurrent_executions = 10
  code_signing_config_arn        = aws_lambda_code_signing_config.lambda_code_signing.arn

  environment {
    variables = {
      REGION                  = var.aws_region
      DYNAMODB_TABLE          = local.aws_dynamodb_table_name
      DYNAMODB_ORIGINAL_INDEX = local.aws_dynamodb_original_index
      DYNAMODB_TIME_INDEX     = local.aws_dynamodb_time_index
      S3_BUCKET               = local.aws_s3_bucket_name
      SQS_QUEUE               = local.aws_sqs_queue_name
      WEBHOOK_URL             = local.webhook_url
    }
  }

  tracing_config {
    mode = "Active"
  }

  depends_on = [
    aws_cloudwatch_log_group.email_receive_logs,
    aws_iam_role_policy_attachment.lambda_logs,
    aws_iam_role_policy_attachment.lambda_dynamodb_s3
  ]
}

resource "aws_lambda_function" "functions" {
  #checkov:skip=CKV_AWS_116: TODO: add SQS for DLQ
  for_each                       = tomap(local.lambda_functions)
  function_name                  = "${local.project_name_env}-${each.key}"
  filename                       = "bin/${each.value.function}.zip"
  handler                        = "bootstrap"
  runtime                        = "provided.al2023"
  role                           = aws_iam_role.lambda_exec_role.arn
  source_code_hash               = filebase64sha256("bin/${each.value.function}.zip")
  reserved_concurrent_executions = 10
  code_signing_config_arn        = aws_lambda_code_signing_config.lambda_code_signing.arn

  environment {
    variables = {
      REGION                  = var.aws_region
      DYNAMODB_TABLE          = local.aws_dynamodb_table_name
      DYNAMODB_ORIGINAL_INDEX = local.aws_dynamodb_original_index
      DYNAMODB_TIME_INDEX     = local.aws_dynamodb_time_index
      S3_BUCKET               = local.aws_s3_bucket_name
      SQS_QUEUE               = local.aws_sqs_queue_name
      WEBHOOK_URL             = local.webhook_url
    }
  }

  tracing_config {
    mode = "Active"
  }

  depends_on = [
    aws_cloudwatch_log_group.function_logs,
    aws_iam_role_policy_attachment.lambda_logs,
    aws_iam_role_policy_attachment.lambda_dynamodb_s3
  ]
}

resource "aws_apigatewayv2_integration" "integrations" {
  for_each               = tomap(local.lambda_functions)
  api_id                 = aws_apigatewayv2_api.mailbox_api.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_function.functions[each.key].invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "routes" {
  for_each           = tomap(local.lambda_functions)
  api_id             = aws_apigatewayv2_api.mailbox_api.id
  route_key          = "${each.value.httpMethod} ${each.value.httpPath}"
  target             = "integrations/${aws_apigatewayv2_integration.integrations[each.key].id}"
  authorization_type = "AWS_IAM"
}

resource "aws_lambda_permission" "apigw_invoke" {
  for_each      = tomap(local.lambda_functions)
  statement_id  = "AllowAPIGatewayInvoke-${each.key}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.functions[each.key].function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.mailbox_api.execution_arn}/*/${each.value.httpMethod}${each.value.arnPath}"
}

#trivy:ignore:AWS-0024
resource "aws_dynamodb_table" "mailbox_table" {
  #checkov:skip=CKV_AWS_28

  # Only managed when no existing table is supplied
  count = var.aws_dynamodb_table_override == "" ? 1 : 0

  name           = local.aws_dynamodb_table_name
  billing_mode   = "PROVISIONED"
  read_capacity  = 3
  write_capacity = 1
  hash_key       = "MessageID"
  attribute {
    name = "MessageID"
    type = "S"
  }
  attribute {
    name = "TypeYearMonth"
    type = "S"
  }
  attribute {
    name = "DateTime"
    type = "S"
  }
  attribute {
    name = "OriginalMessageID"
    type = "S"
  }

  global_secondary_index {
    name = local.aws_dynamodb_time_index
    key_schema {
      attribute_name = "TypeYearMonth"
      key_type       = "HASH"
    }
    key_schema {
      attribute_name = "DateTime"
      key_type       = "RANGE"
    }
    projection_type = "INCLUDE"
    non_key_attributes = [
      "Subject",
      "From",
      "To",
      "Unread",
      "TrashedTime",
      "ThreadID",
      "IsThreadLatest"
    ]
    read_capacity  = 3
    write_capacity = 1
  }

  global_secondary_index {
    name = local.aws_dynamodb_original_index
    key_schema {
      attribute_name = "OriginalMessageID"
      key_type       = "HASH"
    }
    projection_type = "KEYS_ONLY"
    read_capacity   = 3
    write_capacity  = 1
  }
}

resource "aws_sqs_queue" "notifications" {
  name = local.aws_sqs_queue_name

  # 14-day retention
  message_retention_seconds  = 1209600
  visibility_timeout_seconds = 30
  sqs_managed_sse_enabled    = true
}

resource "aws_lambda_permission" "ses_invoke_email_receive" {
  count = var.ses_receipt_rule_name != "" ? 1 : 0

  statement_id   = "AllowSESInvoke"
  action         = "lambda:InvokeFunction"
  function_name  = aws_lambda_function.email_receive.function_name
  principal      = "ses.amazonaws.com"
  source_account = data.aws_caller_identity.current.account_id
}

# Only one rule set can be active per region, so manage the rule, not the set
resource "aws_ses_receipt_rule" "receive" {
  count = var.ses_receipt_rule_name != "" ? 1 : 0

  name          = var.ses_receipt_rule_name
  rule_set_name = var.ses_receipt_rule_set_name
  enabled       = true
  scan_enabled  = true
  tls_policy    = "Optional"

  s3_action {
    bucket_name = local.aws_s3_bucket_name
    position    = 1
  }

  lambda_action {
    function_arn    = aws_lambda_function.email_receive.arn
    invocation_type = "Event"
    position        = 2
  }

  depends_on = [aws_lambda_permission.ses_invoke_email_receive]
}
