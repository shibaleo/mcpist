# Cloudflare Module
# Manages: KV Namespaces, DNS Records, Worker Configuration

terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

variable "account_id" {
  description = "Cloudflare Account ID"
  type        = string
}

variable "zone_id" {
  description = "Cloudflare Zone ID"
  type        = string
}

variable "domain" {
  description = "Domain name"
  type        = string
}

variable "environment" {
  description = "Environment (dev/prod)"
  type        = string
}

# KV Namespace for Rate Limiting
resource "cloudflare_workers_kv_namespace" "rate_limit" {
  account_id = var.account_id
  title      = "mcpist-rate-limit-${var.environment}"
}

# KV Namespace for Health State
resource "cloudflare_workers_kv_namespace" "health_state" {
  account_id = var.account_id
  title      = "mcpist-health-state-${var.environment}"
}

# DNS Record for API Gateway
resource "cloudflare_record" "api" {
  zone_id = var.zone_id
  name    = var.environment == "prod" ? "api" : "api-${var.environment}"
  content = "mcpist-gateway.${var.account_id}.workers.dev"
  type    = "CNAME"
  proxied = true
}

output "rate_limit_kv_id" {
  value = cloudflare_workers_kv_namespace.rate_limit.id
}

output "health_state_kv_id" {
  value = cloudflare_workers_kv_namespace.health_state.id
}

output "api_hostname" {
  value = cloudflare_record.api.hostname
}
