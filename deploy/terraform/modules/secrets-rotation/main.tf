# AWS Secrets Manager Rotation Configuration
# Enables automated 90-day rotation for database credentials
# Uses a Lambda rotation function attached to the RDS instance

resource "aws_secretsmanager_secret_rotation" "postgres" {
  secret_id          = var.postgres_secret_id
  rotation_lambda_arn = aws_lambda_function.rotation.arn

  rotation_rules {
    automatically_after_days = var.rotation_interval_days
  }
}

# Lambda function for RDS credential rotation
resource "aws_lambda_function" "rotation" {
  function_name = "${var.project_name}-${var.environment}-secret-rotation"
  description   = "Rotates RDS credentials for ${var.project_name}"

  runtime     = "python3.11"
  handler     = "index.lambda_handler"
  timeout     = 60
  memory_size = 128

  # AWS-provided rotation function for RDS
  # In production, use the AWS Serverless Application Repository version
  filename         = "${path.module}/rotation_lambda.zip"
  source_code_hash = filebase64sha256("${path.module}/rotation_lambda.zip")

  role = aws_iam_role.rotation.arn

  environment {
    variables = {
      SECRETS_MANAGER_ENDPOINT = "https://secretsmanager.${data.aws_region.current.name}.amazonaws.com"
    }
  }

  vpc_config {
    subnet_ids         = var.private_subnet_ids
    security_group_ids = [aws_security_group.rotation.id]
  }

  tags = var.tags
}

# Allow Secrets Manager to invoke the rotation Lambda
resource "aws_lambda_permission" "rotation" {
  statement_id  = "AllowSecretsManagerInvocation"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.rotation.function_name
  principal     = "secretsmanager.amazonaws.com"
}

# IAM Role for rotation Lambda
resource "aws_iam_role" "rotation" {
  name = "${var.project_name}-${var.environment}-rotation-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "rotation" {
  name = "${var.project_name}-${var.environment}-rotation-policy"
  role = aws_iam_role.rotation.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SecretsManagerAccess"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:PutSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [var.postgres_secret_id]
      },
      {
        Sid    = "RDSAccess"
        Effect = "Allow"
        Action = [
          "rds:DescribeDBInstances",
          "rds:ModifyDBInstance"
        ]
        Resource = [var.rds_instance_arn]
      },
      {
        Sid    = "VPCAccess"
        Effect = "Allow"
        Action = [
          "ec2:CreateNetworkInterface",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DeleteNetworkInterface"
        ]
        Resource = "*"
      },
      {
        Sid    = "CloudWatchLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })
}

# Security group for rotation Lambda
resource "aws_security_group" "rotation" {
  name        = "${var.project_name}-${var.environment}-rotation-sg"
  description = "Security group for secrets rotation Lambda"
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

data "aws_region" "current" {}
