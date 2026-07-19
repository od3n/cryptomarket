terraform {
  backend "s3" {
    bucket         = "cryptomarket-staging-terraform-state"
    key            = "staging/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "cryptomarket-staging-terraform-lock"
    encrypt        = true
  }
}
