locals {
  name_prefix = "${var.project_name}-${var.environment}"
  common_tags = merge(var.tags, {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  })
}

# Application secrets placeholder
# In production, these would be populated via CI/CD or manual rotation
resource "aws_secretsmanager_secret" "app_config" {
  name        = "${local.name_prefix}/app-config"
  description = "Application configuration secrets for ${local.name_prefix}"
  kms_key_id  = var.kms_key_id

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "app_config" {
  secret_id = aws_secretsmanager_secret.app_config.id
  secret_string = jsonencode({
    REDIS_PASSWORD = ""
    API_KEYS       = {}
  })
}

# Redis auth token secret
resource "aws_secretsmanager_secret" "redis_auth" {
  name        = "${local.name_prefix}/redis-auth"
  description = "Redis authentication token for ${local.name_prefix}"
  kms_key_id  = var.kms_key_id

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "redis_auth" {
  secret_id     = aws_secretsmanager_secret.redis_auth.id
  secret_string = jsonencode({ auth_token = "" })
}

# Secret rotation configuration (documentation)
# Rotation is handled via:
# 1. RDS: Automatic rotation via Secrets Manager (if enabled)
# 2. Redis: Manual rotation during maintenance windows
# 3. API Keys: Rotated via provider dashboards, updated in Secrets Manager
