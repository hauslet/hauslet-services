# Terraform Outputs for Hauslet GCP Deployment

output "scheduler_jobs" {
  description = "Cloud Scheduler job names and schedules"
  value = {
    media_cleanup = {
      name     = google_cloud_scheduler_job.media_cleanup.name
      schedule = google_cloud_scheduler_job.media_cleanup.schedule
      endpoint = google_cloud_scheduler_job.media_cleanup.http_target[0].uri
    }
    booking_expiry = {
      name     = google_cloud_scheduler_job.booking_expiry.name
      schedule = google_cloud_scheduler_job.booking_expiry.schedule
      endpoint = google_cloud_scheduler_job.booking_expiry.http_target[0].uri
    }
    booking_completion = {
      name     = google_cloud_scheduler_job.booking_completion.name
      schedule = google_cloud_scheduler_job.booking_completion.schedule
      endpoint = google_cloud_scheduler_job.booking_completion.http_target[0].uri
    }
    payout_process = {
      name     = google_cloud_scheduler_job.payout_process.name
      schedule = google_cloud_scheduler_job.payout_process.schedule
      endpoint = google_cloud_scheduler_job.payout_process.http_target[0].uri
    }
    disbursement_retry = {
      name     = google_cloud_scheduler_job.disbursement_retry.name
      schedule = google_cloud_scheduler_job.disbursement_retry.schedule
      endpoint = google_cloud_scheduler_job.disbursement_retry.http_target[0].uri
    }
    penalty_debt_collection = {
      name     = google_cloud_scheduler_job.penalty_debt_collection.name
      schedule = google_cloud_scheduler_job.penalty_debt_collection.schedule
      endpoint = google_cloud_scheduler_job.penalty_debt_collection.http_target[0].uri
    }
    finance_reconciliation = {
      name     = google_cloud_scheduler_job.finance_reconciliation.name
      schedule = google_cloud_scheduler_job.finance_reconciliation.schedule
      endpoint = google_cloud_scheduler_job.finance_reconciliation.http_target[0].uri
    }
    review_publish_standoffs = {
      name     = google_cloud_scheduler_job.review_publish_standoffs.name
      schedule = google_cloud_scheduler_job.review_publish_standoffs.schedule
      endpoint = google_cloud_scheduler_job.review_publish_standoffs.http_target[0].uri
    }
    review_send_reminders = {
      name     = google_cloud_scheduler_job.review_send_reminders.name
      schedule = google_cloud_scheduler_job.review_send_reminders.schedule
      endpoint = google_cloud_scheduler_job.review_send_reminders.http_target[0].uri
    }
  }
}
