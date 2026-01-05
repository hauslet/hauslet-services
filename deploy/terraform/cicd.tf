# CI/CD Infrastructure for Hauslet Services
# Artifact Registry, Cloud Build triggers, and service accounts

# Enable required APIs
resource "google_project_service" "artifactregistry" {
  project = var.project_id
  service = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudbuild" {
  project = var.project_id
  service = "cloudbuild.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "run" {
  project = var.project_id
  service = "run.googleapis.com"
  disable_on_destroy = false
}

# Artifact Registry repository for Docker images
resource "google_artifact_registry_repository" "hauslet" {
  project       = var.project_id
  location      = var.region
  repository_id = "hauslet"
  description   = "Hauslet Docker images for API and Worker services"
  format        = "DOCKER"

  cleanup_policies {
    id     = "keep-recent-20"
    action = "DELETE"

    condition {
      tag_state  = "UNTAGGED"
      older_than = "2592000s"  # 30 days
    }
  }

  cleanup_policies {
    id     = "keep-tagged-90-days"
    action = "KEEP"

    most_recent_versions {
      keep_count = 20
    }
  }

  depends_on = [
    google_project_service.artifactregistry
  ]
}

# Storage bucket for build artifacts
resource "google_storage_bucket" "build_artifacts" {
  name     = "${var.project_id}-build-artifacts"
  location = var.region
  project  = var.project_id

  uniform_bucket_level_access = true

  lifecycle_rule {
    condition {
      age = 90  # Delete after 90 days
    }
    action {
      type = "Delete"
    }
  }

  versioning {
    enabled = false
  }
}

# Service Accounts for Cloud Run services
# API Service Account
resource "google_service_account" "api" {
  project      = var.project_id
  account_id   = "hauslet-api-sa"
  display_name = "Hauslet API Service Account"
  description  = "Service account for Hauslet API Cloud Run service"
}

# Worker Service Account
resource "google_service_account" "worker" {
  project      = var.project_id
  account_id   = "hauslet-worker-sa"
  display_name = "Hauslet Worker Service Account"
  description  = "Service account for Hauslet Worker Cloud Run service"
}

# Staging Service Accounts
resource "google_service_account" "api_staging" {
  project      = var.project_id
  account_id   = "hauslet-api-staging-sa"
  display_name = "Hauslet API Staging Service Account"
  description  = "Service account for Hauslet API staging environment"
}

resource "google_service_account" "worker_staging" {
  project      = var.project_id
  account_id   = "hauslet-worker-staging-sa"
  display_name = "Hauslet Worker Staging Service Account"
  description  = "Service account for Hauslet Worker staging environment"
}

# IAM Permissions for API Service Account
resource "google_project_iam_member" "api_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.api.email}"
}

resource "google_project_iam_member" "api_secretmanager" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.api.email}"
}

resource "google_project_iam_member" "api_cloudtasks" {
  project = var.project_id
  role    = "roles/cloudtasks.enqueuer"
  member  = "serviceAccount:${google_service_account.api.email}"
}

resource "google_project_iam_member" "api_storage" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# IAM Permissions for Worker Service Account
resource "google_project_iam_member" "worker_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

resource "google_project_iam_member" "worker_secretmanager" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

resource "google_project_iam_member" "worker_storage" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Cloud Run Invoker for Worker (allows Cloud Tasks/Scheduler to invoke)
resource "google_cloud_run_service_iam_member" "worker_invoker" {
  project  = var.project_id
  location = var.region
  service  = var.worker_service_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.worker.email}"
}

# Cloud Build Service Account permissions
data "google_project" "project" {
  project_id = var.project_id
}

# Grant Cloud Build permission to deploy to Cloud Run
resource "google_project_iam_member" "cloudbuild_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"
}

# Grant Cloud Build permission to use service accounts
resource "google_service_account_iam_member" "cloudbuild_api_sa" {
  service_account_id = google_service_account.api.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"
}

resource "google_service_account_iam_member" "cloudbuild_worker_sa" {
  service_account_id = google_service_account.worker.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"
}

# Grant Cloud Build access to Artifact Registry
resource "google_artifact_registry_repository_iam_member" "cloudbuild_writer" {
  project    = var.project_id
  location   = google_artifact_registry_repository.hauslet.location
  repository = google_artifact_registry_repository.hauslet.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"
}

# Cloud Build Triggers (created via gcloud or console, documented in outputs)
# Note: GitHub connection must be set up manually first via:
# gcloud builds connections create github hauslet-github --region=us-central1

# Outputs
output "artifact_registry_url" {
  description = "Artifact Registry repository URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.hauslet.repository_id}"
}

output "api_service_account_email" {
  description = "API service account email"
  value       = google_service_account.api.email
}

output "worker_service_account_email" {
  description = "Worker service account email"
  value       = google_service_account.worker.email
}

output "build_artifacts_bucket" {
  description = "Cloud Storage bucket for build artifacts"
  value       = google_storage_bucket.build_artifacts.name
}
