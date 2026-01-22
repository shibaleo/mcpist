# Vercel Module
# Manages: Project, Environment Variables, Custom Domain

terraform {
  required_providers {
    vercel = {
      source  = "vercel/vercel"
      version = "~> 1.0"
    }
  }
}

variable "team_id" {
  description = "Vercel Team ID"
  type        = string
  default     = null
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "mcpist-console"
}

variable "git_repo" {
  description = "Git repository URL"
  type        = string
}

variable "environment" {
  description = "Environment (dev/prod)"
  type        = string
}

variable "supabase_url" {
  description = "Supabase URL"
  type        = string
}

variable "supabase_publishable_key" {
  description = "Supabase Publishable Key"
  type        = string
  sensitive   = true
}

variable "mcp_server_url" {
  description = "MCP Server URL (Worker)"
  type        = string
}

# Vercel Project
resource "vercel_project" "console" {
  name      = "${var.project_name}-${var.environment}"
  framework = "nextjs"
  team_id   = var.team_id

  git_repository = {
    type = "github"
    repo = var.git_repo
  }

  root_directory = "apps/console"
}

# Environment Variables
resource "vercel_project_environment_variable" "supabase_url" {
  project_id = vercel_project.console.id
  team_id    = var.team_id
  key        = "NEXT_PUBLIC_SUPABASE_URL"
  value      = var.supabase_url
  target     = ["production", "preview"]
}

resource "vercel_project_environment_variable" "supabase_publishable_key" {
  project_id = vercel_project.console.id
  team_id    = var.team_id
  key        = "NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY"
  value      = var.supabase_publishable_key
  target     = ["production", "preview"]
}

resource "vercel_project_environment_variable" "mcp_server_url" {
  project_id = vercel_project.console.id
  team_id    = var.team_id
  key        = "NEXT_PUBLIC_MCP_SERVER_URL"
  value      = var.mcp_server_url
  target     = ["production", "preview"]
}

output "project_id" {
  value = vercel_project.console.id
}

output "deployment_url" {
  value = "https://${var.project_name}-${var.environment}.vercel.app"
}
