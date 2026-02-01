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

output "lambda_dlq_url" {
  description = "URL of the Lambda Dead Letter Queue"
  value       = aws_sqs_queue.lambda_dlq.url
}

output "lambda_dlq_arn" {
  description = "ARN of the Lambda Dead Letter Queue"
  value       = aws_sqs_queue.lambda_dlq.arn
}
