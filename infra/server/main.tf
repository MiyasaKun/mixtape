resource "hcloud_firewall" "production" {
  name = "mixtape-production"

  rule {
    direction   = "in"
    protocol    = "tcp"
    port        = "22"
    source_ips  = ["0.0.0.0/0", "::/0"]
    description = "SSH"
  }

  rule {
    direction   = "in"
    protocol    = "tcp"
    port        = "80"
    source_ips  = ["0.0.0.0/0", "::/0"]
    description = "HTTP for Caddy and ACME"
  }

  rule {
    direction   = "in"
    protocol    = "tcp"
    port        = "443"
    source_ips  = ["0.0.0.0/0", "::/0"]
    description = "HTTPS for Caddy"
  }
}

resource "hcloud_server" "production" {
  name        = var.server_name
  server_type = var.server_type
  location    = var.location
  image       = var.image
  user_data = templatefile("${path.module}/cloud-init.yaml.tftpl", {
    deploy_user      = var.deploy_user
    deploy_principal = var.deploy_principal
    deploy_ca_public = var.deploy_ca_public_key
  })
  firewall_ids = [hcloud_firewall.production.id]

  labels = {
    application = "mixtape"
    environment = "production"
  }

  lifecycle {
    prevent_destroy = true
  }
}
