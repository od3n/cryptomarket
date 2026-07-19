output "certificate_arn" {
  description = "ACM certificate ARN"
  value       = aws_acm_certificate.main.arn
}

output "certificate_domain" {
  description = "Certificate domain name"
  value       = aws_acm_certificate.main.domain_name
}
