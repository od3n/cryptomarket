output "api_role_arn" {
  description = "IAM role ARN for API service"
  value       = aws_iam_role.api.arn
}

output "ingestor_role_arn" {
  description = "IAM role ARN for Ingestor service"
  value       = aws_iam_role.ingestor.arn
}

output "external_secrets_role_arn" {
  description = "IAM role ARN for External Secrets Operator"
  value       = aws_iam_role.external_secrets.arn
}
