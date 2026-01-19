# Render Module
# Manages: Web Service deployment
#
# Note: Render has official Terraform provider

terraform {
  required_providers {
    render = {
      source  = "render-oss/render"
      version = "~> 1.0"
    }
  }
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "mcpist-api"
}

variable "environment" {
  description = "Environment (dev/prod)"
  type        = string
}

variable "region" {
  description = "Deployment region"
  type        = string
  default     = "oregon"
}

variable "docker_image" {
  description = "Docker image URL"
  type        = string
}

variable "supabase_url" {
  description = "Supabase URL"
  type        = string
}

variable "supabase_service_role_key" {
  description = "Supabase Service Role Key"
  type        = string
  sensitive   = true
}

variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "starter"
}

locals {
  full_app_name = "${var.app_name}-${var.environment}"
}

# Render Web Service
resource "render_web_service" "api" {
  name   = local.full_app_name
  region = var.region

  runtime_source = {
    docker = {
      image_url = var.docker_image
    }
  }

  env_vars = {
    SUPABASE_URL              = { value = var.supabase_url }
    SUPABASE_SERVICE_ROLE_KEY = { value = var.supabase_service_role_key }
    INSTANCE_ID               = { value = "render" }
    INSTANCE_REGION           = { value = var.region }
    PORT                      = { value = "8089" }
  }

  plan = var.instance_type
}

output "service_id" {
  value = render_web_service.api.id
}

output "service_url" {
  value = "https://${render_web_service.api.slug}.onrender.com"
}
