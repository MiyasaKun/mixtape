provider "docker" {
  host = "ssh://${var.deploy_user}@${var.deploy_host}:${var.deploy_port}"

  ssh_opts = [
    "-i", var.ssh_identity_file,
    "-o", "CertificateFile=${var.ssh_certificate_file}",
    "-o", "IdentitiesOnly=yes",
    "-o", "UserKnownHostsFile=${var.ssh_known_hosts_file}",
    "-o", "StrictHostKeyChecking=yes",
    "-o", "BatchMode=yes",
  ]

  registry_auth {
    address  = var.registry_address
    username = var.registry_username
    password = var.registry_password
  }
}
