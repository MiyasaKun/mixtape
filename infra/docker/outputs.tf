output "application_container" {
  description = "Application Docker container name."
  value       = docker_container.app.name
}

output "application_url" {
  description = "Application URL. Uses HTTP until a domain is configured."
  value       = var.domain == "" ? "http://${var.deploy_host}" : "https://${var.domain}"
}
