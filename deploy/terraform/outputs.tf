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
  }
}
