variable "project_name" {
  type    = string
  default = "cryptomarket"
}

variable "environment" {
  type = string
}

variable "eks_cluster_name" {
  description = "EKS cluster name for IRSA"
  type        = string
}

variable "eks_oidc_provider_arn" {
  description = "EKS OIDC provider ARN"
  type        = string
}

variable "eks_oidc_issuer_url" {
  description = "EKS OIDC issuer URL"
  type        = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
