variable "caddy_image" {
  description = "Pinned Caddy image reference."
  type        = string
  default     = "caddy:2-alpine"
}

variable "db_name" {
  description = "PostgreSQL database name."
  type        = string
  default     = "discord_bot"
}

variable "db_password" {
  description = "PostgreSQL application password."
  type        = string
  sensitive   = true
}

variable "db_user" {
  description = "PostgreSQL application user."
  type        = string
  default     = "user"
}

variable "deploy_host" {
  description = "IP address or hostname of the production Docker host."
  type        = string
}

variable "deploy_port" {
  description = "SSH port of the production Docker host."
  type        = number
  default     = 22
}

variable "deploy_user" {
  description = "SSH user created by cloud-init for Docker deployments."
  type        = string
  default     = "mixtape-deploy"
}

variable "discord_app_id" {
  description = "Discord application ID."
  type        = string
}

variable "discord_bot_token" {
  description = "Discord bot token."
  type        = string
  sensitive   = true
}

variable "domain" {
  description = "Public application hostname. Leave blank until DNS is configured."
  type        = string
  default     = ""

  validation {
    condition     = var.domain == "" || !strcontains(var.domain, "://")
    error_message = "domain must be a hostname without a URL scheme."
  }
}

variable "guild_id" {
  description = "Optional Discord guild ID."
  type        = string
  default     = ""
}

variable "image" {
  description = "Immutable application image reference, including tag or digest."
  type        = string
  default     = "registry.gitlab.com/miyasakun/mixtape"
}

variable "postgres_image" {
  description = "Pinned PostgreSQL image reference."
  type        = string
  default     = "postgres:18-alpine3.23"
}

variable "registry_address" {
  description = "Container registry hostname."
  type        = string
  default     = "registry.gitlab.com"
}

variable "registry_password" {
  description = "Container registry password or deploy token."
  type        = string
  sensitive   = true
}

variable "registry_username" {
  description = "Container registry username."
  type        = string
}

variable "spotify_client_id" {
  description = "Spotify client ID."
  type        = string
  default     = ""
}

variable "spotify_client_secret" {
  description = "Spotify client secret."
  type        = string
  sensitive   = true
  default     = ""
}

variable "spotify_redirect_uri" {
  description = "Spotify OAuth callback URL. Leave blank to derive it from domain."
  type        = string
  default     = ""
}

variable "ssh_certificate_file" {
  description = "Path to the signed OpenSSH user certificate."
  type        = string
}

variable "ssh_identity_file" {
  description = "Path to the SSH private key used by the deployment certificate."
  type        = string
}

variable "ssh_known_hosts_file" {
  description = "Path to the verified known_hosts file for the Docker host."
  type        = string
}

variable "valkey_image" {
  description = "Pinned Valkey image reference."
  type        = string
  default     = "valkey/valkey:alpine"
}
