terraform {
  backend "s3" {
    bucket         = "cryptomarket-prod-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "cryptomarket-prod-terraform-lock"
    encrypt        = true
  }
}
