locals {
  app_name = "mixtape"

  caddy_site = var.domain == "" ? ":80" : var.domain

  spotify_redirect_uri = var.spotify_redirect_uri != "" ? var.spotify_redirect_uri : (
    var.domain == "" ? "http://localhost:3000/api/auth/spotify/callback" : "https://${var.domain}/api/auth/spotify/callback"
  )

  database_url = "postgresql://${urlencode(var.db_user)}:${urlencode(var.db_password)}@postgres:5432/${urlencode(var.db_name)}?sslmode=disable"

  caddyfile = <<-EOT
    ${local.caddy_site} {
      reverse_proxy app:3000
    }
  EOT
}
