terraform {
  backend "s3" {
    bucket         = "cryptomarket-dev-terraform-state"
    key            = "dev/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "cryptomarket-dev-terraform-lock"
    encrypt        = true
  }
}
