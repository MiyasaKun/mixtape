# Production Deployment

Production infrastructure is managed by Terraform and GitLab CI.

## Architecture

- Hetzner Cloud `cx33` server in Falkenstein.
- Debian 13 with Docker installed by cloud-init.
- Caddy reverse proxy.
- Mixtape, PostgreSQL, and Valkey containers on a private Docker network.
- PostgreSQL, Valkey, and Caddy data stored in Docker volumes.
- Terraform state stored in GitLab-managed HTTP state.

The server and Docker stack use separate Terraform roots:

- `infra/server`: Hetzner server and cloud-init, state name `production-server`.
- `infra/docker`: Remote Docker resources, state name `production`.

## GitLab Variables

Create these protected GitLab variables before running the pipeline:

- `HCLOUD_TOKEN`: Hetzner API token.
- `REGISTRY_DEPLOY_USER`: GitLab deploy-token username with `read_registry` permission.
- `REGISTRY_DEPLOY_PASSWORD`: GitLab deploy-token password with `read_registry` permission.
- `TF_VAR_deploy_ca_public_key`: contents of the SSH user CA public key.
- `DEPLOY_SSH_PRIVATE_KEY`: file variable containing the deployment private key.
- `DEPLOY_SSH_CERTIFICATE`: file variable containing the signed deployment certificate.
- `DEPLOY_KNOWN_HOSTS`: file variable containing the verified server host key.
- `TF_VAR_db_password`: PostgreSQL password.
- `TF_VAR_discord_bot_token`: Discord bot token.
- `TF_VAR_discord_app_id`: Discord application ID.
- `TF_VAR_guild_id`: optional Discord guild ID.
- `TF_VAR_spotify_client_id`: optional Spotify client ID.
- `TF_VAR_spotify_client_secret`: optional Spotify client secret.

Leave `TF_VAR_domain` unset or empty until DNS is configured. The application will then be served over HTTP at the server IP. When a domain is ready, set `TF_VAR_domain` to the hostname without `https://`, point DNS to the server, and run the deployment again. Caddy will obtain the HTTPS certificate automatically.

## Create SSH Credentials

Run these commands on a trusted administration machine. Do not commit any generated files.

```sh
umask 077
ssh-keygen -t ed25519 -f mixtape-deploy-ca -C mixtape-deploy-ca
ssh-keygen -t ed25519 -f mixtape-deploy -C mixtape-production
ssh-keygen -s mixtape-deploy-ca \
  -I mixtape-production \
  -n mixtape-deployer \
  -V +90d \
  mixtape-deploy.pub
```

Use the following files as GitLab variables:

- `mixtape-deploy-ca.pub` -> `TF_VAR_deploy_ca_public_key`
- `mixtape-deploy` -> `DEPLOY_SSH_PRIVATE_KEY`
- `mixtape-deploy-cert.pub` -> `DEPLOY_SSH_CERTIFICATE`

Keep `mixtape-deploy-ca` offline. It is the CA private key and is not needed by GitLab CI.

## First Deployment

1. Run `terraform-server-plan` and review the plan.
2. Run the manual `terraform-server-apply` job.
3. Obtain the server IP from the Terraform output or Hetzner Console.
4. Verify the server host-key fingerprint out of band.
5. Generate a known-hosts entry and store it as the `DEPLOY_KNOWN_HOSTS` file variable:

   ```sh
   ssh-keyscan -H SERVER_IP
   ```

6. Run `terraform-deploy-plan` and review the plan.
7. Run the manual `terraform-deploy-apply` job.

The deployment job reads the server IP from the `production-server` Terraform state, so it does not need a manually maintained `DEPLOY_HOST` variable.

## Domain Activation

After DNS is ready:

1. Create an `A` record for the domain pointing to the Hetzner IPv4 address.
2. Add an `AAAA` record if IPv6 will be used.
3. Set protected GitLab variable `TF_VAR_domain` to the hostname.
4. Update Spotify’s callback URL to `https://DOMAIN/api/auth/spotify/callback` if Spotify integration is enabled.
5. Run the Terraform plan and manual apply jobs.

Caddy will use ports 80 and 443 for HTTP validation, HTTPS, and application traffic. PostgreSQL and Valkey are not exposed through the Hetzner firewall.
