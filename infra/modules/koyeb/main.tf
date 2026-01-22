# Koyeb Module
# Manages: App, Service, Environment Variables

terraform {
  required_providers {
    koyeb = {
      source  = "koyeb/koyeb"
      version = "~> 0.1"
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
  default     = "nano"
}

variable "regions" {
  description = "Deployment regions"
  type        = list(string)
  default     = ["fra"]
}

# Koyeb App
resource "koyeb_app" "api" {
  name = "${var.app_name}-${var.environment}"
}

# Koyeb Service
resource "koyeb_service" "api" {
  app_name = koyeb_app.api.name

  definition {
    name = "api"

    instance_types {
      type = var.instance_type
    }

    regions = var.regions

    scalings {
      min = 1
      max = 2
    }

    ports {
      port     = 8089
      protocol = "http"
    }

    routes {
      path = "/"
      port = 8089
    }

    health_checks {
      http {
        port = 8089
        path = "/health"
      }
    }

    docker {
      image = var.docker_image
    }

    env {
      key   = "PORT"
      value = "8089"
    }

    env {
      key   = "INSTANCE_ID"
      value = "koyeb"
    }

    env {
      key   = "INSTANCE_REGION"
      value = var.regions[0]
    }

    env {
      key   = "SUPABASE_URL"
      value = var.supabase_url
    }

    env {
      key    = "SUPABASE_SERVICE_ROLE_KEY"
      value  = var.supabase_service_role_key
      secret = true
    }
  }
}

output "app_id" {
  value = koyeb_app.api.id
}

output "service_url" {
  value = "https://${koyeb_app.api.name}.koyeb.app"
}
