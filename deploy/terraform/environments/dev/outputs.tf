output "vpc_id" {
  description = "VPC ID"
  value       = module.networking.vpc_id
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds.endpoint
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = module.elasticache.primary_endpoint_address
}

output "terraform_state_bucket" {
  description = "Terraform state bucket"
  value       = module.s3.terraform_state_bucket
}

output "backups_bucket" {
  description = "Backups bucket"
  value       = module.s3.backups_bucket
}
