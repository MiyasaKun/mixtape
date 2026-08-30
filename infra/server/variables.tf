variable "server_name" {
  description = "Hetzner server name."
  type        = string
  default     = "mixtape-production"
}

variable "server_type" {
  description = "Hetzner server type."
  type        = string
  default     = "cx33"
}

variable "location" {
  description = "Hetzner location."
  type        = string
  default     = "fsn1"
}

variable "image" {
  description = "Hetzner operating system image."
  type        = string
  default     = "debian-13"
}

variable "deploy_user" {
  description = "SSH deployment user created on the server."
  type        = string
  default     = "mixtape-deploy"
}

variable "deploy_principal" {
  description = "SSH certificate principal accepted by the deployment user."
  type        = string
  default     = "mixtape-deployer"
}

variable "deploy_ca_public_key" {
  description = "OpenSSH user CA public key trusted by the server."
  type        = string
}
