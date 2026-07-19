variable "project_name" {
  type    = string
  default = "cryptomarket"
}

variable "environment" {
  type = string
}

variable "domain_name" {
  type = string
}

variable "subject_alternative_names" {
  description = "SANs for the certificate"
  type        = list(string)
  default     = []
}

variable "zone_id" {
  description = "Route53 zone ID for DNS validation"
  type        = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
