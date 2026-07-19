output "terraform_state_bucket" {
  description = "Terraform state bucket name"
  value       = aws_s3_bucket.terraform_state.id
}

output "backups_bucket" {
  description = "Backups bucket name"
  value       = aws_s3_bucket.backups.id
}

output "artifacts_bucket" {
  description = "Artifacts bucket name"
  value       = aws_s3_bucket.artifacts.id
}

output "dynamodb_lock_table" {
  description = "DynamoDB lock table name"
  value       = aws_dynamodb_table.terraform_lock.name
}
