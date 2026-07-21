variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "cryptomarket"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
}

variable "rate_limit_threshold" {
  description = "Maximum requests per 5-minute period per IP before blocking"
  type        = number
  default     = 2000
}

variable "blocked_countries" {
  description = "List of ISO 3166-1 alpha-2 country codes to block"
  type        = list(string)
  default     = []
}

variable "log_destination_arn" {
  description = "ARN of the S3 bucket or Kinesis Firehose for WAF logs"
  type        = string
}

variable "alarm_sns_topic_arn" {
  description = "SNS topic ARN for WAF alarms (empty to disable)"
  type        = string
  default     = ""
}

variable "blocked_requests_alarm_threshold" {
  description = "Number of blocked requests in 10min to trigger alarm"
  type        = number
  default     = 500
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default     = {}
}
