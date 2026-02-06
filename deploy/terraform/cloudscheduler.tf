# Cloud Scheduler Jobs for Hauslet Worker
# Replaces manual ticker functions in cmd/worker/setup/handlers.go

# 1. Media Cleanup Scheduler (every 15 minutes)
resource "google_cloud_scheduler_job" "media_cleanup" {
  name        = "media-cleanup-scheduler"
  description = "Triggers media cleanup job every 15 minutes"
  schedule    = "*/15 * * * *" # Every 15 minutes
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 3
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
  schedule    = "*/2 * * * *" # Every 2 minutes
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 3
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
  schedule    = "0 * * * *" # Every hour at minute 0
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 3
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

# 4. Booking Check-In/Out Scheduler (every hour)
resource "google_cloud_scheduler_job" "booking_checkin_out" {
  name        = "booking-checkin-out-scheduler"
  description = "Auto-populates booking check-in/out timestamps every hour"
  schedule    = "0 * * * *" # Every hour at minute 0
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/booking/checkin-out"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching BookingCheckInOutJob
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

# 5. Payout Processing Scheduler (every hour)
resource "google_cloud_scheduler_job" "payout_process" {
  name        = "payout-process-scheduler"
  description = "Processes pending payouts every hour"
  schedule    = "0 * * * *" # Every hour at minute 0
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 2 # Lower retry for financial operations
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

# 6. Disbursement Retry Scheduler (every 15 minutes)
resource "google_cloud_scheduler_job" "disbursement_retry" {
  name        = "disbursement-retry-scheduler"
  description = "Retries failed disbursements every 15 minutes"
  schedule    = "*/15 * * * *" # Every 15 minutes
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 2 # Lower retry for financial operations
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

# 6b. Penalty Debt Collection Scheduler (every 30 minutes)
resource "google_cloud_scheduler_job" "penalty_debt_collection" {
  name        = "penalty-debt-collection-scheduler"
  description = "Collects outstanding host cancellation penalty debts every 30 minutes"
  schedule    = "*/30 * * * *" # Every 30 minutes
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 2
    min_backoff_duration = "10s"
    max_backoff_duration = "300s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/finance/penalty-debt/collection"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching PenaltyDebtCollectionJob
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

# 7. Financial Reconciliation Scheduler (daily at 2 AM UTC)
resource "google_cloud_scheduler_job" "finance_reconciliation" {
  name        = "finance-reconciliation-scheduler"
  description = "Runs daily financial reconciliation at 2 AM UTC"
  schedule    = "0 2 * * *" # Daily at 2:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 1 # Single retry for reconciliation
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

# 8. Review Standoff Publishing Scheduler (daily at midnight UTC)
resource "google_cloud_scheduler_job" "review_publish_standoffs" {
  name        = "review-publish-standoffs-scheduler"
  description = "Publishes reviews stuck in standoff after 14 days"
  schedule    = "0 0 * * *" # Daily at midnight UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 2
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

# 9. Review Reminder Scheduler (daily at 10 AM UTC)
resource "google_cloud_scheduler_job" "review_send_reminders" {
  name        = "review-send-reminders-scheduler"
  description = "Sends review reminders to users approaching deadline"
  schedule    = "0 10 * * *" # Daily at 10:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 2
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

# 10. Promotion Expiry Scheduler (daily at 1 AM UTC)
resource "google_cloud_scheduler_job" "promotion_expiry" {
  name        = "promotion-expiry-scheduler"
  description = "Expires promotions that have passed their end date"
  schedule    = "0 1 * * *" # Daily at 1:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 2
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/promotion/expiry"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching PromotionExpiryJob
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

# 11. Subscription Billing Scheduler (daily at 3 AM UTC)
resource "google_cloud_scheduler_job" "subscription_billing" {
  name        = "subscription-billing-scheduler"
  description = "Processes billing for subscriptions due for renewal"
  schedule    = "0 3 * * *" # Daily at 3:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 2 # Lower retry for billing operations
    min_backoff_duration = "10s"
    max_backoff_duration = "300s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/promotion/billing"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching SubscriptionBillingJob
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

# 14. Calendar Showing Reminders (1 hour before)
resource "google_cloud_scheduler_job" "calendar_showing_reminders_1h" {
  name        = "calendar-showing-reminders-1h"
  description = "Sends showing reminders 1 hour before scheduled time"
  schedule    = "*/15 * * * *" # Every 15 minutes
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/calendar/showing/reminders"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Send reminders for showings 60 minutes away
    body = base64encode(jsonencode({
      reminder_minutes = 60
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

# 15. Calendar Showing Reminders (24 hours before)
resource "google_cloud_scheduler_job" "calendar_showing_reminders_24h" {
  name        = "calendar-showing-reminders-24h"
  description = "Sends showing reminders 24 hours before scheduled time"
  schedule    = "0 */4 * * *" # Every 4 hours
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/calendar/showing/reminders"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Send reminders for showings 1440 minutes (24h) away
    body = base64encode(jsonencode({
      reminder_minutes = 1440
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

# 16. Calendar Open House Reminders (1 hour before)
resource "google_cloud_scheduler_job" "calendar_open_house_reminders_1h" {
  name        = "calendar-open-house-reminders-1h"
  description = "Sends open house reminders 1 hour before event"
  schedule    = "*/15 * * * *" # Every 15 minutes
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/calendar/open-house/reminders"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Send reminders for open houses 60 minutes away
    body = base64encode(jsonencode({
      reminder_minutes = 60
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

# 17. Calendar Open House Reminders (24 hours before)
resource "google_cloud_scheduler_job" "calendar_open_house_reminders_24h" {
  name        = "calendar-open-house-reminders-24h"
  description = "Sends open house reminders 24 hours before event"
  schedule    = "0 */4 * * *" # Every 4 hours
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/calendar/open-house/reminders"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Send reminders for open houses 1440 minutes (24h) away
    body = base64encode(jsonencode({
      reminder_minutes = 1440
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

# 18. Interactions Batch Writer (every 2 minutes)
resource "google_cloud_scheduler_job" "interactions_batch_writer" {
  name        = "interactions-batch-writer"
  description = "Processes interactions from Redis queue to database"
  schedule    = "*/2 * * * *" # Every 2 minutes
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/interactions/batch"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload - handler reads from Redis queue
    # No parameters needed
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

# 19. Interactions Aggregator (every hour)
resource "google_cloud_scheduler_job" "interactions_aggregator" {
  name        = "interactions-aggregator"
  description = "Aggregates raw interactions into analytics (hourly rollups)"
  schedule    = "5 * * * *" # Every hour at 5 minutes past (gives batch writer time to process)
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "10s"
    max_backoff_duration = "120s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/interactions/aggregate"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload - aggregates previous hour by default
    # period_type: "hour" or "day"
    # period_start: optional timestamp (defaults to previous hour)
    body = base64encode(jsonencode({
      period_type = "hour"
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

# 16. Verification Reconciliation Scheduler (every 6 hours)
resource "google_cloud_scheduler_job" "verification_reconciliation" {
  name        = "verification-reconciliation-scheduler"
  description = "Reconciles verified sessions with profile module every 6 hours"
  schedule    = "0 */6 * * *" # Every 6 hours at minute 0
  time_zone   = "UTC"
  region      = "europe-west1" # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 2
    min_backoff_duration = "10s"
    max_backoff_duration = "120s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/verification/reconciliation"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching ReconciliationJob
    body = base64encode(jsonencode({
      batch_size = 100
      timestamp  = "" # Handler will use current time
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

# 20. Conversation Cleanup Scheduler (daily at 3 AM UTC)
resource "google_cloud_scheduler_job" "conversation_cleanup" {
  name        = "conversation-cleanup-scheduler"
  description = "Archives stale conversations and deletes old archived ones"
  schedule    = "0 3 * * *" # Daily at 3:00 AM UTC
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 2
    min_backoff_duration = "10s"
    max_backoff_duration = "120s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/conversation/cleanup"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching ConversationCleanupJob
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

# 21. Listing Embedding Generator (daily at midnight)
resource "google_cloud_scheduler_job" "listing_embedding" {
  name        = "listing-embedding-scheduler"
  description = "Generates embeddings for listings without them daily"
  schedule    = "0 0 * * *" # Daily at midnight
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count          = 3
    min_backoff_duration = "10s"
    max_backoff_duration = "300s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/listing/embedding"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload matching GenerateEmbeddingsJob
    body = base64encode(jsonencode({
      batch_size = 50
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
