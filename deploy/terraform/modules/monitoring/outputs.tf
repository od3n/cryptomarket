output "eks_log_group_name" {
  description = "EKS CloudWatch log group name"
  value       = aws_cloudwatch_log_group.eks.name
}

output "application_log_group_name" {
  description = "Application CloudWatch log group name"
  value       = aws_cloudwatch_log_group.application.name
}
