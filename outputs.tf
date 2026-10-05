
output "api_endpoint" {
  description = "Endpoint URL of the mailbox API Gateway"
  value       = aws_apigatewayv2_api.mailbox_api.api_endpoint
  sensitive   = true
}

output "api_id" {
  description = "ID of the mailbox API Gateway"
  value       = aws_apigatewayv2_api.mailbox_api.id
  sensitive   = true
}

output "api_arn" {
  description = "ARN of the mailbox API Gateway"
  value       = aws_apigatewayv2_api.mailbox_api.arn
  sensitive   = true
}

output "sqs_queue_url" {
  description = "URL of the email notification queue"
  value       = aws_sqs_queue.notifications.url
  sensitive   = true
}

output "sqs_queue_arn" {
  description = "ARN of the email notification queue"
  value       = aws_sqs_queue.notifications.arn
  sensitive   = true
}

output "email_receive_dlq_url" {
  description = "URL of the dead-letter queue for failed email receives"
  value       = aws_sqs_queue.email_receive_dlq.url
  sensitive   = true
}

output "email_receive_dlq_arn" {
  description = "ARN of the dead-letter queue for failed email receives"
  value       = aws_sqs_queue.email_receive_dlq.arn
  sensitive   = true
}
