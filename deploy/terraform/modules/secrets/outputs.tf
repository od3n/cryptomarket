output "app_config_secret_arn" {
  description = "ARN of the app config secret"
  value       = aws_secretsmanager_secret.app_config.arn
}

output "redis_auth_secret_arn" {
  description = "ARN of the Redis auth secret"
  value       = aws_secretsmanager_secret.redis_auth.arn
}
