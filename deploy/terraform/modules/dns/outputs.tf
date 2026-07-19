output "zone_id" {
  description = "Route53 hosted zone ID"
  value       = local.zone_id
}

output "domain_name" {
  description = "Domain name"
  value       = var.domain_name
}
