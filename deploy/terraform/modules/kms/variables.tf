variable "project_name" {
  type    = string
  default = "cryptomarket"
}

variable "environment" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
