# MCPist Development Environment
# Minimal configuration for development/staging

terraform {
  required_version = ">= 1.6.0"

  # Local backend for dev
  backend "local" {
    path = "terraform.tfstate"
  }
}

# Variables
variable "cloudflare_api_token" {
  type      = string
  sensitive = true
}

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "domain" {
  type    = string
  default = "mcpist.io"
}

locals {
  environment = "dev"
}

# Providers
provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Only Cloudflare for dev (KV namespaces for testing)
module "cloudflare" {
  source = "../../modules/cloudflare"

  account_id  = var.cloudflare_account_id
  zone_id     = var.cloudflare_zone_id
  domain      = var.domain
  environment = local.environment
}

# Outputs
output "rate_limit_kv_id" {
  value = module.cloudflare.rate_limit_kv_id
}

output "health_state_kv_id" {
  value = module.cloudflare.health_state_kv_id
}

output "api_hostname" {
  value = module.cloudflare.api_hostname
}
