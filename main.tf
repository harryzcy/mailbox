provider "aws" {
  region = var.aws_region
}

resource "aws_apigatewayv2_api" "mailbox_api" {
  name          = "${var.project_name}-api"
  protocol_type = "HTTP"
}

resource "aws_cloudwatch_log_group" "mailbox_api_access_logs" {
  name = "/aws/apigateway/${var.project_name}-mailbox-api-access-logs"
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


output "api_endpoint" {
  description = "Endpoint URL of the mailbox API Gateway"
  value       = aws_apigatewayv2_api.mailbox_api.api_endpoint
}

output "api_id" {
  description = "ID of the mailbox API Gateway"
  value       = aws_apigatewayv2_api.mailbox_api.id
}

output "api_arn" {
  description = "ARN of the mailbox API Gateway"
  value       = aws_apigatewayv2_api.mailbox_api.arn
}
