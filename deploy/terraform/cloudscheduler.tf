# Cloud Scheduler Jobs for Hauslet Worker
# Replaces manual ticker functions in cmd/worker/setup/handlers.go

# 1. Media Cleanup Scheduler (every 15 minutes)
resource "google_cloud_scheduler_job" "media_cleanup" {
  name        = "media-cleanup-scheduler"
  description = "Triggers media cleanup job every 15 minutes"
  schedule    = "*/15 * * * *"  # Every 15 minutes
  time_zone   = "UTC"
  region      = var.region

  retry_config {
    retry_count = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "1m"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/media/cleanup"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching ListingMediaCleanupJob
    body = base64encode(jsonencode({
      older_than_minutes = 120
      limit              = 200
    }))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 2. Booking Expiry Check Scheduler (every 2 minutes)
resource "google_cloud_scheduler_job" "booking_expiry" {
  name        = "booking-expiry-scheduler"
  description = "Checks for expired bookings every 2 minutes"
  schedule    = "*/2 * * * *"  # Every 2 minutes
  time_zone   = "UTC"
  region      = var.region

  retry_config {
    retry_count = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "1m"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/booking/expiry"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching BookingExpiryCheckJob
    # Note: check_time will be set by the job handler to current time
    body = base64encode(jsonencode({
      check_time = "dynamic"  # Handler will use time.Now()
    }))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 3. Payout Processing Scheduler (every hour)
resource "google_cloud_scheduler_job" "payout_process" {
  name        = "payout-process-scheduler"
  description = "Processes pending payouts every hour"
  schedule    = "0 * * * *"  # Every hour at minute 0
  time_zone   = "UTC"
  region      = var.region

  retry_config {
    retry_count = 2  # Lower retry for financial operations
    min_backoff_duration = "10s"
    max_backoff_duration = "5m"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/finance/payout/process"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching ProcessPayoutsJob
    body = base64encode(jsonencode({
      process_time = "dynamic"  # Handler will use time.Now()
    }))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 4. Disbursement Retry Scheduler (every 15 minutes)
resource "google_cloud_scheduler_job" "disbursement_retry" {
  name        = "disbursement-retry-scheduler"
  description = "Retries failed disbursements every 15 minutes"
  schedule    = "*/15 * * * *"  # Every 15 minutes
  time_zone   = "UTC"
  region      = var.region

  retry_config {
    retry_count = 2  # Lower retry for financial operations
    min_backoff_duration = "10s"
    max_backoff_duration = "5m"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/finance/payout/retry"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching RetryDisbursementsJob
    body = base64encode(jsonencode({
      retry_time = "dynamic"  # Handler will use time.Now()
    }))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# Enable Cloud Scheduler API
resource "google_project_service" "cloudscheduler" {
  project = var.project_id
  service = "cloudscheduler.googleapis.com"

  disable_on_destroy = false
}

# Grant scheduler permission to invoke worker
resource "google_cloud_run_service_iam_member" "scheduler_invoker" {
  project  = var.project_id
  location = var.region
  service  = var.worker_service_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.worker_service_account}"
}
