variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "cryptomarket"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
}

variable "postgres_secret_id" {
  description = "ARN of the Secrets Manager secret for PostgreSQL credentials"
  type        = string
}

variable "rds_instance_arn" {
  description = "ARN of the RDS instance to rotate credentials for"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for the rotation Lambda"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for the rotation Lambda VPC config"
  type        = list(string)
}

variable "rotation_interval_days" {
  description = "Number of days between automatic rotations"
  type        = number
  default     = 90
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default     = {}
}
