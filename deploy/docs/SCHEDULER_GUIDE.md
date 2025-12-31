# Cloud Scheduler Guide for Hauslet Services

Complete guide to managing Cloud Scheduler jobs for Hauslet background tasks.

## Table of Contents

- [Overview](#overview)
- [Current Jobs](#current-jobs)
- [Adding New Jobs](#adding-new-jobs)
- [Managing Jobs](#managing-jobs)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Cron Patterns](#cron-patterns)

## Overview

Cloud Scheduler triggers periodic background jobs by making HTTP POST requests to your Worker service endpoints.

### Architecture

```
┌──────────────────┐
│                  │
│  Cloud Scheduler │  Cron schedule (e.g., every 15 min)
│  (europe-west1)  │
│                  │
└────────┬─────────┘
         │ HTTP POST with OIDC token
         │
         ▼
┌──────────────────┐
│                  │
│  Cloud Run       │  Receives request
│  Worker Service  │  Processes job
│  (europe-north1) │  Returns 200 OK
│                  │
└──────────────────┘
```

### Why Cloud Scheduler?

**Before (Manual Tickers):**
```go
// Worker had to run continuously
go StartPeriodicCleanup()  // Runs in background
go StartBookingExpiryCheck()
// Cost: Worker must scale to min 1 instance always
```

**After (Cloud Scheduler):**
```go
// Worker scales to 0 when idle
// Scheduler triggers endpoints on schedule
// Cost: Only pay when jobs run
```

**Savings:**
- Worker can scale to 0 between jobs
- No background goroutines consuming resources
- Only ~$0.40/month for scheduler jobs

## Current Jobs

| Job | Schedule | Frequency | Endpoint | Purpose |
|-----|----------|-----------|----------|---------|
| `media-cleanup-scheduler` | `*/15 * * * *` | Every 15 min | `/tasks/media/cleanup` | Delete orphaned media files |
| `booking-expiry-scheduler` | `*/2 * * * *` | Every 2 min | `/tasks/booking/expiry` | Cancel expired pending bookings |
| `payout-process-scheduler` | `0 * * * *` | Every hour | `/tasks/finance/payout/process` | Process pending host payouts |
| `disbursement-retry-scheduler` | `*/15 * * * *` | Every 15 min | `/tasks/finance/payout/retry` | Retry failed disbursements |

### Job Details

**Media Cleanup:**
- Deletes media files older than 2 hours that aren't attached to listings
- Prevents storage costs from temporary uploads
- Handler: `internal/transport/worker/handlers/listing/media_cleanup.go`

**Booking Expiry:**
- Checks for bookings in PENDING status that expired
- Cancels booking and frees calendar slot
- Handler: `internal/transport/worker/handlers/booking/expiry.go`

**Payout Processing:**
- Finds bookings ready for payout (after escrow period)
- Initiates transfers to host bank accounts
- Handler: `internal/transport/worker/handlers/finance/payout_process.go`

**Disbursement Retry:**
- Retries failed payout disbursements
- Uses exponential backoff
- Handler: `internal/transport/worker/handlers/finance/disbursement_retry.go`

## Adding New Jobs

### Step 1: Add to Terraform

Edit `deploy/terraform/cloudscheduler.tf`:

```hcl
# Example: Weekly analytics report
resource "google_cloud_scheduler_job" "weekly_analytics" {
  name        = "weekly-analytics-scheduler"
  description = "Generate weekly analytics report every Monday at 9am"
  schedule    = "0 9 * * 1"  # 9am UTC on Mondays
  time_zone   = "UTC"
  region      = "europe-west1"  # Cloud Scheduler not available in europe-north1

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/analytics/weekly"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    # Job payload (base64 encoded JSON)
    body = base64encode(jsonencode({
      report_type = "weekly"
      include_charts = true
    }))

    # OIDC authentication
    oidc_token {
      service_account_email = var.worker_service_account
      audience              = var.worker_url
    }
  }

  depends_on = [
    google_project_service.cloudscheduler
  ]
}
```

### Step 2: Apply Terraform

```bash
cd deploy/terraform

# Review changes
terraform plan

# Apply
terraform apply
```

### Step 3: Create Worker Handler

Create handler in your worker code:

```go
// internal/transport/worker/handlers/analytics/weekly.go
package analytics

type WeeklyAnalyticsHandler struct {
    analyticsService *analytics.Service
    log              *slog.Logger 
    queueName        string
}

func NewWeeklyAnalyticsHandler(
    svc *analytics.Service,
    log *slog.Logger ,
    queueName string,
) *WeeklyAnalyticsHandler {
    return &WeeklyAnalyticsHandler{
        analyticsService: svc,
        log:              log,
        queueName:        queueName,
    }
}

func (h *WeeklyAnalyticsHandler) QueueName() string {
    return h.queueName
}

func (h *WeeklyAnalyticsHandler) Handle(ctx context.Context, payload []byte) error {
    var job WeeklyAnalyticsJob
    if err := json.Unmarshal(payload, &job); err != nil {
        return fmt.Errorf("unmarshal job: %w", err)
    }

    h.log.Info(" Processing weekly analytics report: %+v", job)

    // Your business logic here
    if err := h.analyticsService.GenerateWeeklyReport(ctx, job); err != nil {
        return fmt.Errorf("generate report: %w", err)
    }

    return nil
}

type WeeklyAnalyticsJob struct {
    ReportType     string `json:"report_type"`
    IncludeCharts  bool   `json:"include_charts"`
}
```

### Step 4: Register Handler

In `cmd/worker/setup/handlers.go`:

```go
func RegisterHandlers(infra *Infrastructure, cfg *config.GlobalConfig, log *slog.Logger ) *queue.Registry {
    registry := queue.NewRegistry()
    qCfg := cfg.YAML.Queue.Subjects

    // ... existing handlers ...

    // Weekly analytics handler
    if hasWeeklyAnalytics := qCfg["weekly_analytics"] != ""; hasWeeklyAnalytics {
        analyticsRepo := analytics.NewRepository(infra.DB)
        analyticsSvc := analytics.NewService(analyticsRepo, log)
        h := analyticsHandler.NewWeeklyAnalyticsHandler(
            analyticsSvc,
            log,
            qCfg["weekly_analytics"],
        )
        registry.Register(h)
    }

    return registry
}
```

### Step 5: Add Queue Configuration

In `config/defaults/queue.yaml`:

```yaml
queue:
  subjects:
    # ... existing queues ...
    weekly_analytics: "weekly-analytics-queue"
```

### Step 6: Deploy & Test

```bash
# Deploy updated worker
cd deploy
./deploy-staging.sh

# Test manually
gcloud scheduler jobs run weekly-analytics-scheduler \
  --location=europe-west1

# Check logs
gcloud run services logs read hauslet-worker-staging \
  --region=europe-north1 \
  --limit=20
```

## Managing Jobs

### List All Jobs

```bash
gcloud scheduler jobs list --location=europe-west1
```

### Describe Specific Job

```bash
gcloud scheduler jobs describe media-cleanup-scheduler \
  --location=europe-west1
```

### Manually Trigger Job

```bash
# Trigger immediately (useful for testing)
gcloud scheduler jobs run media-cleanup-scheduler \
  --location=europe-west1

# Check if it succeeded
echo $?  # 0 = success
```

### Update Job Schedule

Edit `cloudscheduler.tf` and change the `schedule`:

```hcl
resource "google_cloud_scheduler_job" "media_cleanup" {
  schedule = "*/30 * * * *"  # Changed from 15 to 30 minutes
  # ...
}
```

Apply changes:
```bash
terraform apply
```

### Pause Job

```bash
# Pause job (stop it from running)
gcloud scheduler jobs pause media-cleanup-scheduler \
  --location=europe-west1

# Resume job
gcloud scheduler jobs resume media-cleanup-scheduler \
  --location=europe-west1
```

### Delete Job

```bash
# Via gcloud
gcloud scheduler jobs delete media-cleanup-scheduler \
  --location=europe-west1

# Or via Terraform (recommended)
# 1. Remove from cloudscheduler.tf
# 2. terraform apply
```

## Monitoring

### View Job Status

```bash
# List jobs with status
gcloud scheduler jobs list \
  --location=europe-west1 \
  --format="table(name,schedule,state,lastAttemptTime,status)"
```

### Check Job Execution History

```bash
# View scheduler logs
gcloud logging read \
  "resource.type=cloud_scheduler_job AND resource.labels.job_id=media-cleanup-scheduler" \
  --limit=20 \
  --format=json
```

### Monitor Success/Failure Rate

```bash
# Count recent executions
gcloud logging read \
  "resource.type=cloud_scheduler_job" \
  --format="table(timestamp,resource.labels.job_id,httpRequest.status)" \
  --limit=100

# Count failures
gcloud logging read \
  "resource.type=cloud_scheduler_job AND httpRequest.status>=400" \
  --limit=50
```

### View Worker Processing Logs

```bash
# See job processing in worker
gcloud run services logs read hauslet-worker-staging \
  --region=europe-north1 \
  --limit=50

# Filter for specific job
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-worker-staging AND textPayload=~\"media cleanup\"" \
  --limit=20
```

### Set Up Alerts

Create alert for failed jobs:

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="Scheduler Job Failures" \
  --condition-display-name="High Failure Rate" \
  --condition-threshold-value=5 \
  --condition-threshold-duration=300s \
  --condition-filter='resource.type="cloud_scheduler_job" AND metric.type="cloudscheduler.googleapis.com/job/attempt_count" AND metric.labels.response_class="error"'
```

## Troubleshooting

### Job Returns 403 Forbidden

**Cause:** Worker service doesn't allow scheduler to invoke it.

**Solution:**
```bash
gcloud run services add-iam-policy-binding hauslet-worker-staging \
  --region=europe-north1 \
  --member="serviceAccount:hauslet-worker-staging-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"
```

### Job Returns 404 Not Found

**Cause:** Endpoint doesn't exist in worker.

**Solution:**
1. Check endpoint path in scheduler matches worker route
2. Verify handler is registered in worker
3. Check worker logs for routing errors

### Job Returns 500 Internal Server Error

**Cause:** Handler code is failing.

**Solution:**
1. Check worker logs for error details:
   ```bash
   gcloud run services logs read hauslet-worker-staging \
     --region=europe-north1 \
     --log-filter='severity>=ERROR'
   ```
2. Fix bug in handler code
3. Redeploy worker

### Job Never Triggers

**Causes:**
1. Job is paused
2. Schedule is incorrect
3. Region mismatch

**Solutions:**
```bash
# Check if paused
gcloud scheduler jobs describe JOB_NAME --location=europe-west1

# Resume if paused
gcloud scheduler jobs resume JOB_NAME --location=europe-west1

# Verify schedule format
# Use https://crontab.guru/ to validate

# Verify job exists
gcloud scheduler jobs list --location=europe-west1
```

### Job Times Out

**Cause:** Handler takes longer than timeout allows.

**Solution:** Increase timeout in scheduler config:

```hcl
resource "google_cloud_scheduler_job" "slow_job" {
  # ...

  http_target {
    # Increase from default (30s) to 5 minutes
    uri = "${var.worker_url}/tasks/slow/process"

    # Note: Cloud Run timeout must also be increased
    # gcloud run services update hauslet-worker-staging --timeout=600s
  }
}
```

## Cron Patterns

### Format

```
 ┌───────────── minute (0 - 59)
 │ ┌───────────── hour (0 - 23)
 │ │ ┌───────────── day of month (1 - 31)
 │ │ │ ┌───────────── month (1 - 12)
 │ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
 │ │ │ │ │
 * * * * *
```

### Common Patterns

```bash
# Every minute
* * * * *

# Every 5 minutes
*/5 * * * *

# Every 15 minutes
*/15 * * * *

# Every hour at minute 0
0 * * * *

# Every 6 hours
0 */6 * * *

# Daily at midnight
0 0 * * *

# Daily at 9am
0 9 * * *

# Every Monday at 9am
0 9 * * 1

# First day of month at midnight
0 0 1 * *

# Every weekday at 6pm
0 18 * * 1-5

# Every Saturday at noon
0 12 * * 6

# Quarterly (Jan, Apr, Jul, Oct)
0 0 1 1,4,7,10 *

# Twice daily (6am and 6pm)
0 6,18 * * *
```

### Testing Cron Patterns

Use [crontab.guru](https://crontab.guru/) to validate and understand patterns.

## Best Practices

### 1. Use Appropriate Retry Settings

```hcl
retry_config {
  # For non-critical jobs (media cleanup)
  retry_count = 3
  min_backoff_duration = "5s"
  max_backoff_duration = "60s"
}

retry_config {
  # For financial operations (lower retries)
  retry_count = 2
  min_backoff_duration = "10s"
  max_backoff_duration = "300s"
}
```

### 2. Set Reasonable Schedules

- **Too frequent:** Wastes resources, increases costs
- **Too infrequent:** Users experience delays

| Job Type | Good Frequency | Bad Frequency |
|----------|----------------|---------------|
| Cleanup tasks | Every 15-30 min | Every 1 min |
| Financial processing | Hourly or daily | Every 5 min |
| Reports/analytics | Daily or weekly | Hourly |
| Health checks | Every 1-5 min | Every 30 sec |

### 3. Make Handlers Idempotent

Handler should safely handle duplicate calls:

```go
func (h *Handler) Handle(ctx context.Context, payload []byte) error {
    // ✅ GOOD: Check if already processed
    if h.service.AlreadyProcessed(job.ID) {
        h.log.Info(" Job %s already processed, skipping", job.ID)
        return nil
    }

    // Process job...
}
```

### 4. Use Timezone Wisely

```hcl
# UTC (default) - Best for international services
time_zone = "UTC"

# Specific timezone - For regional jobs
time_zone = "America/New_York"
time_zone = "Europe/London"
time_zone = "Asia/Tokyo"

# Find timezone names:
# https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
```

### 5. Monitor Job Performance

Track:
- Success rate
- Execution time
- Retry frequency
- Error types

Set up alerts for:
- Multiple consecutive failures
- Execution time over threshold
- High retry rate

### 6. Document Job Purpose

Always add clear descriptions:

```hcl
resource "google_cloud_scheduler_job" "cleanup" {
  description = "Deletes media files >2hrs old, not attached to listings. Runs every 15min to prevent storage bloat."
  # ...
}
```

### 7. Test Before Production

```bash
# 1. Deploy to staging first
./deploy-staging.sh

# 2. Manually trigger test
gcloud scheduler jobs run JOB_NAME --location=europe-west1

# 3. Verify in logs
gcloud run services logs read hauslet-worker-staging

# 4. Monitor for 24 hours

# 5. Deploy to production
./deploy-production.sh
```

## Cost Optimization

**Cloud Scheduler Pricing:**
- $0.10 per job per month
- First 3 jobs free
- 4 jobs = $0.10/month

**Optimization Tips:**

1. **Combine related jobs:**
   ```hcl
   # Instead of 3 separate cleanup jobs
   # Create 1 job that does all cleanup
   ```

2. **Adjust frequency:**
   ```hcl
   # Change from every 5 min to every 15 min
   schedule = "*/15 * * * *"  # 4x fewer executions
   ```

3. **Use appropriate timeouts:**
   ```hcl
   # Don't set timeout higher than needed
   # Each second costs Cloud Run execution time
   ```

## Reference

- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)
- [Cron Schedule Syntax](https://cloud.google.com/scheduler/docs/configuring/cron-job-schedules)
- [OIDC Authentication](https://cloud.google.com/scheduler/docs/http-target-auth)
- [Terraform Google Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_scheduler_job)
- [Crontab Guru](https://crontab.guru/) - Cron expression validator
