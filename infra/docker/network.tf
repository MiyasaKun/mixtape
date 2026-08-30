resource "docker_network" "mixtape" {
  name = "${local.app_name}-network"
}
