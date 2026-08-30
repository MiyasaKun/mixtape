resource "docker_volume" "caddy_config" {
  name = "${local.app_name}-caddy-config"
}

resource "docker_volume" "caddy_data" {
  name = "${local.app_name}-caddy-data"
}

resource "docker_volume" "postgres" {
  name = "${local.app_name}-postgres-data"
}

resource "docker_volume" "valkey" {
  name = "${local.app_name}-valkey-data"
}
