variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "cryptomarket"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

variable "domain_name" {
  description = "Domain name"
  type        = string
  default     = "staging.cryptomarket.example.com"
}
