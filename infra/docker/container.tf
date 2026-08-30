resource "docker_container" "postgres" {
  name    = "${local.app_name}-postgres"
  image   = docker_image.postgres.image_id
  restart = "unless-stopped"

  env = [
    "POSTGRES_USER=${var.db_user}",
    "POSTGRES_PASSWORD=${var.db_password}",
    "POSTGRES_DB=${var.db_name}",
  ]

  mounts {
    target = "/var/lib/postgresql"
    source = docker_volume.postgres.name
    type   = "volume"
  }

  networks_advanced {
    name    = docker_network.mixtape.name
    aliases = ["postgres"]
  }

  healthcheck {
    test         = ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
    interval     = "10s"
    timeout      = "5s"
    retries      = 5
    start_period = "10s"
  }
}

resource "docker_container" "valkey" {
  name    = "${local.app_name}-valkey"
  image   = docker_image.valkey.image_id
  restart = "unless-stopped"

  mounts {
    target = "/data"
    source = docker_volume.valkey.name
    type   = "volume"
  }

  networks_advanced {
    name    = docker_network.mixtape.name
    aliases = ["valkey"]
  }

  healthcheck {
    test         = ["CMD", "valkey-cli", "ping"]
    interval     = "10s"
    timeout      = "5s"
    retries      = 5
    start_period = "5s"
  }
}

resource "docker_container" "app" {
  name         = "${local.app_name}-app"
  image        = docker_image.app.image_id
  restart      = "unless-stopped"
  wait         = true
  wait_timeout = 180

  env = [
    "PORT=:3000",
    "ENV=production",
    "DISCORD_BOT_TOKEN=${var.discord_bot_token}",
    "DISCORD_APP_ID=${var.discord_app_id}",
    "GUILD_ID=${var.guild_id}",
    "SPOTIFY_CLIENT_ID=${var.spotify_client_id}",
    "SPOTIFY_CLIENT_SECRET=${var.spotify_client_secret}",
    "SPOTIFY_REDIRECT_URI=${local.spotify_redirect_uri}",
    "DATABASE_URL=${local.database_url}",
    "REDIS_URL=redis://valkey:6379",
  ]

  networks_advanced {
    name    = docker_network.mixtape.name
    aliases = ["app"]
  }

  healthcheck {
    test         = ["CMD-SHELL", "wget -q -O - http://127.0.0.1:3000/api/health || exit 1"]
    interval     = "30s"
    timeout      = "5s"
    retries      = 3
    start_period = "40s"
  }

  depends_on = [
    docker_container.postgres,
    docker_container.valkey,
  ]
}

resource "docker_container" "caddy" {
  name         = "${local.app_name}-caddy"
  image        = docker_image.caddy.image_id
  restart      = "unless-stopped"
  wait         = true
  wait_timeout = 120

  upload {
    file        = "/etc/caddy/Caddyfile"
    content     = local.caddyfile
    permissions = "0644"
  }

  mounts {
    target = "/data"
    source = docker_volume.caddy_data.name
    type   = "volume"
  }

  mounts {
    target = "/config"
    source = docker_volume.caddy_config.name
    type   = "volume"
  }

  ports {
    internal = 80
    external = 80
  }

  ports {
    internal = 443
    external = 443
  }

  networks_advanced {
    name    = docker_network.mixtape.name
    aliases = ["caddy"]
  }

  healthcheck {
    test         = ["CMD", "caddy", "validate", "--config", "/etc/caddy/Caddyfile"]
    interval     = "30s"
    timeout      = "5s"
    retries      = 3
    start_period = "10s"
  }

  depends_on = [docker_container.app]
}
