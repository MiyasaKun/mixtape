data "docker_registry_image" "app" {
  name = var.image
}

data "docker_registry_image" "caddy" {
  name = var.caddy_image
}

data "docker_registry_image" "postgres" {
  name = var.postgres_image
}

data "docker_registry_image" "valkey" {
  name = var.valkey_image
}

resource "docker_image" "app" {
  name          = data.docker_registry_image.app.name
  pull_triggers = [data.docker_registry_image.app.sha256_digest]
  keep_locally  = true
}

resource "docker_image" "caddy" {
  name          = data.docker_registry_image.caddy.name
  pull_triggers = [data.docker_registry_image.caddy.sha256_digest]
  keep_locally  = true
}

resource "docker_image" "postgres" {
  name          = data.docker_registry_image.postgres.name
  pull_triggers = [data.docker_registry_image.postgres.sha256_digest]
  keep_locally  = true
}

resource "docker_image" "valkey" {
  name          = data.docker_registry_image.valkey.name
  pull_triggers = [data.docker_registry_image.valkey.sha256_digest]
  keep_locally  = true
}
