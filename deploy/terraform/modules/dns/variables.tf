variable "project_name" {
  type    = string
  default = "cryptomarket"
}

variable "environment" {
  type = string
}

variable "domain_name" {
  description = "Root domain name"
  type        = string
  default     = "cryptomarket.example.com"
}

variable "create_hosted_zone" {
  description = "Whether to create a Route53 hosted zone"
  type        = bool
  default     = false
}

variable "tags" {
  type    = map(string)
  default = {}
}
