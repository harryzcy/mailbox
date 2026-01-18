provider "aws" {
  region = var.aws_region
}

resource "aws_apigatewayv2_api" "mailbox_api" {
  name          = "${var.project_name}-api"
  protocol_type = "HTTP"
}

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
