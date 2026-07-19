terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

module "networking" {
  source = "../../modules/networking"

  project_name         = var.project_name
  environment          = var.environment
  vpc_cidr             = "10.2.0.0/16"
  availability_zones   = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
  public_subnet_cidrs  = ["10.2.1.0/24", "10.2.2.0/24", "10.2.3.0/24"]
  private_subnet_cidrs = ["10.2.10.0/24", "10.2.11.0/24", "10.2.12.0/24"]
  tags                 = local.common_tags
}

module "eks" {
  source = "../../modules/eks"

  project_name        = var.project_name
  environment         = var.environment
  vpc_id              = module.networking.vpc_id
  private_subnet_ids  = module.networking.private_subnet_ids
  cluster_version     = "1.29"
  node_instance_types = ["t3.xlarge"]
  node_desired_size   = 4
  node_min_size       = 3
  node_max_size       = 12
  tags                = local.common_tags
}

module "kms" {
  source = "../../modules/kms"

  project_name = var.project_name
  environment  = var.environment
  tags         = local.common_tags
}

module "rds" {
  source = "../../modules/rds"

  project_name            = var.project_name
  environment             = var.environment
  vpc_id                  = module.networking.vpc_id
  private_subnet_ids      = module.networking.private_subnet_ids
  instance_class          = "db.r6g.large"
  allocated_storage       = 100
  multi_az                = true
  backup_retention_period = 35
  deletion_protection     = true
  eks_security_group_id   = module.eks.cluster_security_group_id
  tags                    = local.common_tags
}

module "elasticache" {
  source = "../../modules/elasticache"

  project_name          = var.project_name
  environment           = var.environment
  vpc_id                = module.networking.vpc_id
  private_subnet_ids    = module.networking.private_subnet_ids
  node_type             = "cache.r6g.large"
  num_cache_nodes       = 3
  eks_security_group_id = module.eks.cluster_security_group_id
  tags                  = local.common_tags
}

module "s3" {
  source = "../../modules/s3"

  project_name = var.project_name
  environment  = var.environment
  tags         = local.common_tags
}

module "secrets" {
  source = "../../modules/secrets"

  project_name = var.project_name
  environment  = var.environment
  kms_key_id   = module.kms.key_id
  tags         = local.common_tags
}

module "monitoring" {
  source = "../../modules/monitoring"

  project_name     = var.project_name
  environment      = var.environment
  eks_cluster_name = module.eks.cluster_name
  tags             = local.common_tags
}
