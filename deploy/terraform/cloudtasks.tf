# Cloud Tasks Queues for Hauslet Services
# These queues handle asynchronous job processing

# Enable Cloud Tasks API
resource "google_project_service" "cloudtasks" {
  project = var.project_id
  service = "cloudtasks.googleapis.com"

  disable_on_destroy = false
}

# Email Queue - For sending emails asynchronously
resource "google_cloud_tasks_queue" "email" {
  name     = "email-queue"
  location = "europe-west2"  # Cloud Tasks location

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 5
    max_retry_duration = "3600s"  # 1 hour
    min_backoff        = "5s"
    max_backoff        = "300s"   # 5 minutes
    max_doublings      = 5
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Media Thumbnail Queue - For generating image thumbnails
resource "google_cloud_tasks_queue" "media_thumbnail" {
  name     = "media-thumbnail-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 20
    max_concurrent_dispatches = 10
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "600s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 3
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Media Cleanup Queue - For deleting orphaned media
resource "google_cloud_tasks_queue" "media_cleanup" {
  name     = "media-cleanup-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "300s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 2
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# AI Moderation Queue - For content moderation
resource "google_cloud_tasks_queue" "ai_moderation" {
  name     = "ai-moderation-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 5
    max_concurrent_dispatches = 3
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "600s"
    min_backoff        = "10s"
    max_backoff        = "120s"
    max_doublings      = 3
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Booking Expiry Queue - For processing booking expirations
resource "google_cloud_tasks_queue" "booking_expiry" {
  name     = "booking-expiry-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "300s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 2
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Booking Completion Queue - For completing eligible bookings
resource "google_cloud_tasks_queue" "booking_completion" {
  name     = "booking-completion-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "300s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 2
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Booking Check-In/Out Queue - For auto-populating stay timestamps
resource "google_cloud_tasks_queue" "booking_checkin_out" {
  name     = "booking-checkin-out-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "300s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 2
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Booking Refund Queue - For processing refunds
resource "google_cloud_tasks_queue" "booking_refund" {
  name     = "booking-refund-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 5
    max_concurrent_dispatches = 3
  }

  retry_config {
    max_attempts       = 5
    max_retry_duration = "3600s"
    min_backoff        = "10s"
    max_backoff        = "300s"
    max_doublings      = 4
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Payment Webhook Queue - For processing payment provider webhooks
resource "google_cloud_tasks_queue" "payment_webhook" {
  name     = "payment-webhook-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 20
    max_concurrent_dispatches = 10
  }

  retry_config {
    max_attempts       = 5
    max_retry_duration = "3600s"
    min_backoff        = "10s"
    max_backoff        = "300s"
    max_doublings      = 4
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Payout Processing Queue - For processing host payouts
resource "google_cloud_tasks_queue" "payout_process" {
  name     = "payout-process-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 5
    max_concurrent_dispatches = 2
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "1800s"
    min_backoff        = "30s"
    max_backoff        = "600s"
    max_doublings      = 3
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Payout Retry Queue - For retrying failed payouts
resource "google_cloud_tasks_queue" "payout_retry" {
  name     = "payout-retry-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 5
    max_concurrent_dispatches = 2
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "1800s"
    min_backoff        = "30s"
    max_backoff        = "600s"
    max_doublings      = 3
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Calendar Showing Reminders Queue - For sending viewing appointment reminders
resource "google_cloud_tasks_queue" "calendar_showing_reminders" {
  name     = "calendar-showing-reminders-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "600s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 2
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}

# Calendar Open House Reminders Queue - For sending open house event reminders
resource "google_cloud_tasks_queue" "calendar_open_house_reminders" {
  name     = "calendar-open-house-reminders-queue"
  location = "europe-west2"

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 5
  }

  retry_config {
    max_attempts       = 3
    max_retry_duration = "600s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 2
  }

  depends_on = [
    google_project_service.cloudtasks
  ]
}
