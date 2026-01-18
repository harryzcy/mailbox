provider "aws" {
  region = var.aws_region
}

resource "aws_apigatewayv2_api" "mailbox_api" {
  name          = "${var.project_name}-api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "mailbox_api_default" {
  api_id      = aws_apigatewayv2_api.mailbox_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_stage" "mailbox_api_default" {
  api_id      = aws_apigatewayv2_api.mailbox_api.id
  name        = "dev"
  auto_deploy = true
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
