output "ipv4_address" {
  description = "Public IPv4 address used by the deployment root."
  value       = hcloud_server.production.ipv4_address
}

output "ipv6_address" {
  description = "Public IPv6 address of the production server."
  value       = hcloud_server.production.ipv6_address
}
