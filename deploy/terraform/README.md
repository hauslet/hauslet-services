# Hauslet Cloud Scheduler Terraform Configuration

This directory contains Terraform configuration for deploying Cloud Scheduler jobs that replace the manual ticker functions.

## Overview

**What was replaced:**
- `StartPeriodicCleanup()` → Cloud Scheduler job triggering `/tasks/media/cleanup`
- `StartBookingExpiryCheck()` → Cloud Scheduler job triggering `/tasks/booking/expiry`
- `StartPayoutProcessing()` → Cloud Scheduler job triggering `/tasks/finance/payout/process`
- `StartDisbursementRetry()` → Cloud Scheduler job triggering `/tasks/finance/payout/retry`

**How it works:**
1. Cloud Scheduler jobs run on a cron schedule
2. Each job sends an HTTP POST request to the Worker service endpoint
3. Worker handles the request just like it would handle a Cloud Tasks task
4. OIDC token authentication ensures only Cloud Scheduler can trigger these endpoints

## Prerequisites

1. **Worker Service Deployed**: You must have the Worker Cloud Run service deployed first
2. **Service Account Created**: Worker service account with proper IAM permissions
3. **Terraform Installed**: Version 1.5 or higher
4. **gcloud CLI Authenticated**: `gcloud auth application-default login`

## Setup Instructions

### Step 1: Get Worker Service URL

After deploying your Worker service to Cloud Run, get its URL:

```bash
export WORKER_URL=$(gcloud run services describe hauslet-worker \
  --region=us-central1 \
  --format='value(status.url)')

echo "Worker URL: $WORKER_URL"
```

### Step 2: Create terraform.tfvars

Copy the example file and fill in your values:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
project_id = "hauslet-prod"
region     = "us-central1"

# Replace with your actual Worker URL from Step 1
worker_url             = "https://hauslet-worker-abc123-uc.a.run.app"
worker_service_name    = "hauslet-worker"
worker_service_account = "hauslet-worker-sa@hauslet-prod.iam.gserviceaccount.com"

# API service info
api_service_name    = "hauslet-api"
api_service_account = "hauslet-api-sa@hauslet-prod.iam.gserviceaccount.com"

environment = "production"
```

### Step 3: Initialize Terraform

```bash
cd deploy/terraform
terraform init
```

### Step 4: Plan Deployment

Review what will be created:

```bash
terraform plan
```

You should see:
- ✅ 4 Cloud Scheduler jobs
- ✅ 1 IAM binding (scheduler → worker invoker)
- ✅ 1 API enablement (cloudscheduler.googleapis.com)

### Step 5: Apply Configuration

```bash
terraform apply
```

Type `yes` to confirm.

### Step 6: Verify Deployment

```bash
# List all scheduler jobs
gcloud scheduler jobs list --location=us-central1

# Check specific job details
gcloud scheduler jobs describe media-cleanup-scheduler \
  --location=us-central1
```

## Testing Cloud Scheduler Jobs

### Manual Trigger Test

Trigger each job manually to verify they work:

```bash
# 1. Media Cleanup
gcloud scheduler jobs run media-cleanup-scheduler \
  --location=us-central1

# 2. Booking Expiry Check
gcloud scheduler jobs run booking-expiry-scheduler \
  --location=us-central1

# 3. Payout Processing
gcloud scheduler jobs run payout-process-scheduler \
  --location=us-central1

# 4. Disbursement Retry
gcloud scheduler jobs run disbursement-retry-scheduler \
  --location=us-central1
```

### Check Job Execution Logs

View logs to confirm successful execution:

```bash
# Worker logs (should show task processing)
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-worker" \
  --limit=50 \
  --format=json

# Scheduler logs (should show job triggers)
gcloud logging read \
  "resource.type=cloud_scheduler_job" \
  --limit=20 \
  --format=json
```

### Expected Behavior

After triggering a job, you should see:

1. **Scheduler Logs**: Job execution started/completed
2. **Worker Logs**: HTTP POST received at endpoint (e.g., `/tasks/media/cleanup`)
3. **Task Handler Logs**: Job processing logs (e.g., "Processing media cleanup job")

## Schedules

| Job | Schedule | Frequency | Endpoint |
|-----|----------|-----------|----------|
| Media Cleanup | `*/15 * * * *` | Every 15 minutes | `/tasks/media/cleanup` |
| Booking Expiry | `*/2 * * * *` | Every 2 minutes | `/tasks/booking/expiry` |
| Payout Process | `0 * * * *` | Every hour (on the hour) | `/tasks/finance/payout/process` |
| Disbursement Retry | `*/15 * * * *` | Every 15 minutes | `/tasks/finance/payout/retry` |

## Modifying Schedules

To change a schedule, update the `schedule` field in `cloudscheduler.tf`:

```hcl
resource "google_cloud_scheduler_job" "booking_expiry" {
  schedule = "*/5 * * * *"  # Change to every 5 minutes
  # ...
}
```

Then apply:

```bash
terraform apply
```

## Pausing/Resuming Jobs

```bash
# Pause a job
gcloud scheduler jobs pause media-cleanup-scheduler \
  --location=us-central1

# Resume a job
gcloud scheduler jobs resume media-cleanup-scheduler \
  --location=us-central1
```

## Monitoring

### Set Up Alerts

Create alerting policies for scheduler failures:

```bash
# Example: Alert on job failures
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="Cloud Scheduler Job Failures" \
  --condition-display-name="Job Execution Failed" \
  --condition-threshold-value=1 \
  --condition-threshold-duration=300s
```

### View Job History

```bash
# See recent executions
gcloud scheduler jobs describe media-cleanup-scheduler \
  --location=us-central1 \
  --format="value(status)"
```

## Troubleshooting

### Issue: Job fails with 403 Forbidden

**Cause**: Worker service doesn't allow Cloud Scheduler to invoke it

**Fix**:
```bash
gcloud run services add-iam-policy-binding hauslet-worker \
  --region=us-central1 \
  --member="serviceAccount:hauslet-worker-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"
```

### Issue: Job fails with 404 Not Found

**Cause**: Endpoint path doesn't match worker routes

**Fix**: Verify endpoint paths in `cmd/worker/server.go` match the scheduler job URIs

### Issue: Job succeeds but task doesn't process

**Cause**: Payload format mismatch

**Fix**: Check job payload in `cloudscheduler.tf` matches the expected struct in your handlers

## Cleanup

To remove all scheduler jobs:

```bash
terraform destroy
```

## Cost

Cloud Scheduler pricing: **$0.10 per job per month**

- 4 jobs = **$0.40/month**
- Executions are free (up to 3 jobs per month, then $0.10/job/month)

## Next Steps

After successful deployment:

1. ✅ Monitor scheduler execution for 24 hours
2. ✅ Verify all 4 job types process correctly
3. ✅ Set up alerting for job failures
4. ✅ Remove old ticker code completely (already done in Phase 4)
5. ✅ Update documentation to reflect Cloud Scheduler usage

## References

- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)
- [Cron Schedule Format](https://cloud.google.com/scheduler/docs/configuring/cron-job-schedules)
- [OIDC Authentication](https://cloud.google.com/scheduler/docs/http-target-auth)
