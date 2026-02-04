# Terraform Variables for Hauslet GCP Deployment

variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region for resources"
  type        = string
  default     = "us-central1"
}

variable "worker_url" {
  description = "Cloud Run Worker service URL (e.g., https://hauslet-worker-xxxx-uc.a.run.app)"
  type        = string
}

variable "worker_service_name" {
  description = "Cloud Run Worker service name"
  type        = string
  default     = "hauslet-worker"
}

variable "worker_service_account" {
  description = "Service account email for Worker service"
  type        = string
}

variable "api_service_name" {
  description = "Cloud Run API service name"
  type        = string
  default     = "hauslet-api"
}

variable "api_service_account" {
  description = "Service account email for API service"
  type        = string
}

variable "environment" {
  description = "Environment name (development, staging, production)"
  type        = string
  default     = "production"
}

variable "db_tier" {
  description = "Cloud SQL instance tier"
  type        = string
  default     = "db-custom-2-8192" # 2 vCPU, 8GB RAM
}

variable "redis_memory_size_gb" {
  description = "Memorystore Redis memory size in GB"
  type        = number
  default     = 5
}

variable "domain_name" {
  description = "Domain name for the load balancer (e.g., dev-api.hauslet.com)"
  type        = string
  default     = "dev-api.hauslet.com"
}
