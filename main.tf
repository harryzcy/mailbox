provider "aws" {
  region = var.aws_region
}

locals {
  lambda_functions = {
    emails_list = {
      name       = "emails_list"
      httpMethod = "GET"
      httpPath   = "/emails"
    },
    emails_get = {
      name       = "emails_get"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}"
    },
    emails_getRaw = {
      name       = "emails_getRaw"
      httpMethod = "GET"
      httpPath   = "/emails/{messageID}/raw"
    },
    info = {
      name       = "info"
      httpMethod = "GET"
      httpPath   = "/info"
    }
  }
}

resource "aws_apigatewayv2_api" "mailbox_api" {
  name          = "${var.project_name}-api"
  protocol_type = "HTTP"
}

#trivy:ignore:AVD-AWS-0017
resource "aws_cloudwatch_log_group" "mailbox_api_access_logs" {
  #checkov:skip=CKV_AWS_158: encryption needed for log group
  name              = "/aws/apigateway/${var.project_name}-api-access-logs"
  retention_in_days = 365
}

resource "aws_apigatewayv2_stage" "mailbox_api_default" {
  api_id = aws_apigatewayv2_api.mailbox_api.id
  name   = "$default"
  # auto_deploy = true

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

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

#trivy:ignore:AVD-AWS-0017
resource "aws_cloudwatch_log_group" "function_logs" {
  #checkov:skip=CKV_AWS_158: encryption needed for log group
  for_each          = tomap(local.lambda_functions)
  name              = "/aws/lambda/${local.project_name_env}-${each.key}"
  retention_in_days = 365
}

resource "aws_lambda_function" "functions" {
  #checkov:skip=CKV_AWS_117: VPC access
  #checkov:skip=CKV_AWS_116: TODO: add SQS for DLQ
  #checkov:skip=CKV_AWS_272: TODO: add code signing
  for_each                       = tomap(local.lambda_functions)
  function_name                  = "${local.project_name_env}-${each.key}"
  filename                       = "bin/${each.key}.zip"
  handler                        = "bootstrap"
  runtime                        = "provided.al2023"
  role                           = aws_iam_role.lambda_exec_role.arn
  source_code_hash               = filebase64sha256("bin/${each.key}.zip")
  reserved_concurrent_executions = 10
  tracing_config {
    mode = "Active"
  }

  depends_on = [
    aws_cloudwatch_log_group.function_logs,
    aws_iam_role_policy_attachment.lambda_logs
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
  source_arn    = "${aws_apigatewayv2_api.mailbox_api.execution_arn}/*/${each.value.httpMethod}${each.value.httpPath}"
}
