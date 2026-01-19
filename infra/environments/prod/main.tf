# MCPist Production Environment
# Orchestrates all infrastructure modules

terraform {
  required_version = ">= 1.6.0"

  backend "s3" {
    # Configure your backend
    # bucket = "mcpist-terraform-state"
    # key    = "prod/terraform.tfstate"
    # region = "ap-northeast-1"
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

variable "vercel_api_token" {
  type      = string
  sensitive = true
}

variable "vercel_team_id" {
  type    = string
  default = null
}

variable "git_repo" {
  type    = string
  default = "your-org/mcpist"
}

variable "koyeb_api_token" {
  type      = string
  sensitive = true
}

variable "render_api_token" {
  type      = string
  sensitive = true
}

variable "supabase_access_token" {
  type      = string
  sensitive = true
}

variable "supabase_organization_id" {
  type = string
}

variable "supabase_database_password" {
  type      = string
  sensitive = true
}

variable "docker_image" {
  type    = string
  default = "ghcr.io/your-org/mcpist-api:latest"
}

locals {
  environment = "prod"
}

# Providers
provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

provider "vercel" {
  api_token = var.vercel_api_token
}

provider "koyeb" {
  token = var.koyeb_api_token
}

provider "render" {
  api_key = var.render_api_token
}

# Supabase Module
module "supabase" {
  source = "../../modules/supabase"

  organization_id   = var.supabase_organization_id
  project_name      = "mcpist"
  environment       = local.environment
  region            = "ap-northeast-1"
  database_password = var.supabase_database_password
}

# Cloudflare Module
module "cloudflare" {
  source = "../../modules/cloudflare"

  account_id  = var.cloudflare_account_id
  zone_id     = var.cloudflare_zone_id
  domain      = var.domain
  environment = local.environment
}

# Vercel Module
module "vercel" {
  source = "../../modules/vercel"

  team_id           = var.vercel_team_id
  project_name      = "mcpist-console"
  git_repo          = var.git_repo
  environment       = local.environment
  supabase_url      = module.supabase.api_url
  supabase_anon_key = module.supabase.anon_key
  mcp_server_url    = "https://${module.cloudflare.api_hostname}"
}

# Render Module (Primary Backend)
module "render" {
  source = "../../modules/render"

  app_name                  = "mcpist-api"
  environment               = local.environment
  docker_image              = var.docker_image
  supabase_url              = module.supabase.api_url
  supabase_service_role_key = module.supabase.service_role_key
  instance_type             = "starter"
  region                    = "oregon"
}

# Koyeb Module (Failover Backend)
module "koyeb" {
  source = "../../modules/koyeb"

  app_name                  = "mcpist-api"
  environment               = local.environment
  docker_image              = var.docker_image
  supabase_url              = module.supabase.api_url
  supabase_service_role_key = module.supabase.service_role_key
  instance_type             = "nano"
  regions                   = ["fra"]
}

# Outputs
output "console_url" {
  value = module.vercel.deployment_url
}

output "api_gateway_url" {
  value = "https://${module.cloudflare.api_hostname}"
}

output "backend_primary_url" {
  value = module.render.service_url
}

output "backend_failover_url" {
  value = module.koyeb.service_url
}

output "supabase_url" {
  value = module.supabase.api_url
}
