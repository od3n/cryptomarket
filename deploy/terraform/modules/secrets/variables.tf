variable "project_name" {
  type    = string
  default = "cryptomarket"
}

variable "environment" {
  type = string
}

variable "kms_key_id" {
  description = "KMS key ID for secret encryption"
  type        = string
}

variable "rds_master_secret_arn" {
  description = "ARN of the RDS master user secret"
  type        = string
  default     = ""
}

variable "tags" {
  type    = map(string)
  default = {}
}
