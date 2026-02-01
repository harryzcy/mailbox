provider "aws" {
  region = var.aws_region
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
resource "aws_cloudwatch_log_group" "info_function_logs" {
  #checkov:skip=CKV_AWS_158: encryption needed for log group
  name              = "/aws/lambda/${local.project_name_env}-info"
  retention_in_days = 365
}

resource "aws_lambda_function" "info" {
  function_name    = "${local.project_name_env}-info"
  filename         = "bin/info.zip"
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  role             = aws_iam_role.lambda_exec_role.arn
  source_code_hash = filebase64sha256("bin/info.zip") # Automatically detects changes

  depends_on = [
    aws_cloudwatch_log_group.info_function_logs,
    aws_iam_role_policy_attachment.lambda_logs
  ]
}
