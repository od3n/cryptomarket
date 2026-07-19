variable "project_name" {
  type    = string
  default = "cryptomarket"
}

variable "environment" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "private_subnet_ids" {
  type = list(string)
}

variable "node_type" {
  type    = string
  default = "cache.t3.micro"
}

variable "num_cache_nodes" {
  type    = number
  default = 1
}

variable "eks_security_group_id" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
