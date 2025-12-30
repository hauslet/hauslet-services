# Cloud Scheduler Jobs for Hauslet Worker
# Replaces manual ticker functions in cmd/worker/setup/handlers.go

# 1. Media Cleanup Scheduler (every 15 minutes)
resource "google_cloud_scheduler_job" "media_cleanup" {
  name        = "media-cleanup-scheduler"
  description = "Triggers media cleanup job every 15 minutes"
  schedule    = "*/15 * * * *"  # Every 15 minutes
  time_zone   = "UTC"
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
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
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/booking/expiry"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching BookingExpiryCheckJob
    # Handler will use current time (time.Now())
    body = base64encode(jsonencode({}))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 3. Booking Completion Scheduler (every hour)
resource "google_cloud_scheduler_job" "booking_completion" {
  name        = "booking-completion-scheduler"
  description = "Marks eligible bookings as completed every hour"
  schedule    = "0 * * * *"  # Every hour at minute 0
  time_zone   = "UTC"
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/booking/completion"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching BookingCompletionJob
    body = base64encode(jsonencode({}))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 4. Payout Processing Scheduler (every hour)
resource "google_cloud_scheduler_job" "payout_process" {
  name        = "payout-process-scheduler"
  description = "Processes pending payouts every hour"
  schedule    = "0 * * * *"  # Every hour at minute 0
  time_zone   = "UTC"
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count = 2  # Lower retry for financial operations
    min_backoff_duration = "10s"
    max_backoff_duration = "300s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/finance/payout/process"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching ProcessPayoutsJob
    # Handler will use current time (time.Now())
    body = base64encode(jsonencode({}))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 5. Disbursement Retry Scheduler (every 15 minutes)
resource "google_cloud_scheduler_job" "disbursement_retry" {
  name        = "disbursement-retry-scheduler"
  description = "Retries failed disbursements every 15 minutes"
  schedule    = "*/15 * * * *"  # Every 15 minutes
  time_zone   = "UTC"
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count = 2  # Lower retry for financial operations
    min_backoff_duration = "10s"
    max_backoff_duration = "300s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/finance/payout/retry"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching RetryDisbursementsJob
    # Handler will use current time (time.Now())
    body = base64encode(jsonencode({}))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 6. Financial Reconciliation Scheduler (daily at 2 AM UTC)
resource "google_cloud_scheduler_job" "finance_reconciliation" {
  name        = "finance-reconciliation-scheduler"
  description = "Runs daily financial reconciliation at 2 AM UTC"
  schedule    = "0 2 * * *"  # Daily at 2:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count = 1  # Single retry for reconciliation
    min_backoff_duration = "30s"
    max_backoff_duration = "300s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/finance/reconciliation"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching ReconciliationJob
    # Handler will use current time (time.Now())
    body = base64encode(jsonencode({}))

    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}

# 7. Review Standoff Publishing Scheduler (daily at midnight UTC)
resource "google_cloud_scheduler_job" "review_publish_standoffs" {
  name        = "review-publish-standoffs-scheduler"
  description = "Publishes reviews stuck in standoff after 14 days"
  schedule    = "0 0 * * *"  # Daily at midnight UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count = 2
    min_backoff_duration = "10s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/review/standoff/publish"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching PublishStandoffsJob
    # Uses 14 days threshold (Airbnb standard)
    body = base64encode(jsonencode({
      standoff_threshold_days = 14
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

# 8. Review Reminder Scheduler (daily at 10 AM UTC)
resource "google_cloud_scheduler_job" "review_send_reminders" {
  name        = "review-send-reminders-scheduler"
  description = "Sends review reminders to users approaching deadline"
  schedule    = "0 10 * * *"  # Daily at 10:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count = 2
    min_backoff_duration = "5s"
    max_backoff_duration = "30s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/review/reminders"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching SendReviewRemindersJob
    # Sends reminder 3 days before deadline (day 11 of 14-day window)
    body = base64encode(jsonencode({
      reminder_threshold_days = 3
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
