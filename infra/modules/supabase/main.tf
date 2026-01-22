# Supabase Module
# Manages: Project Settings
# Note: Schema is managed by Supabase CLI (migrations)

terraform {
  required_providers {
    supabase = {
      source  = "supabase/supabase"
      version = "~> 1.0"
    }
  }
}

variable "organization_id" {
  description = "Supabase Organization ID"
  type        = string
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "mcpist"
}

variable "environment" {
  description = "Environment (dev/prod)"
  type        = string
}

variable "region" {
  description = "Project region"
  type        = string
  default     = "ap-northeast-1"
}

variable "database_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

# Supabase Project
resource "supabase_project" "main" {
  organization_id   = var.organization_id
  name              = "${var.project_name}-${var.environment}"
  database_password = var.database_password
  region            = var.region

  lifecycle {
    ignore_changes = [database_password]
  }
}

output "project_id" {
  value = supabase_project.main.id
}

output "project_ref" {
  value = supabase_project.main.id
}

output "api_url" {
  value = "https://${supabase_project.main.id}.supabase.co"
}

output "anon_key" {
  value     = supabase_project.main.anon_key
  sensitive = true
}

output "service_role_key" {
  value     = supabase_project.main.service_role_key
  sensitive = true
}
