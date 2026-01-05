# Hauslet Services Deployment Guide

Complete guide to deploying Hauslet API and Worker services to Google Cloud Platform.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Initial Setup](#initial-setup)
- [Staging Deployment](#staging-deployment)
- [Production Deployment](#production-deployment)
- [Infrastructure Management](#infrastructure-management)
- [Troubleshooting](#troubleshooting)

## Overview

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Google Cloud Platform                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐         ┌──────────────┐                  │
│  │              │         │              │                  │
│  │  Cloud Run   │◄────────│  Cloud Run   │                  │
│  │  API Service │         │  Worker      │                  │
│  │              │         │              │                  │
│  └──────┬───────┘         └──────▲───────┘                  │
│         │                        │                          │
│         │                        │                          │
│         ▼                        │                          │
│  ┌──────────────┐         ┌──────────────┐                  │
│  │              │         │              │                  │
│  │  Cloud SQL   │         │  Cloud       │                  │
│  │  PostgreSQL  │         │  Scheduler   │                  │
│  │              │         │              │                  │
│  └──────────────┘         └──────────────┘                  │
│                                                               │
│  ┌──────────────┐         ┌──────────────┐                  │
│  │              │         │              │                  │
│  │  Memorystore │         │  Secret      │                  │
│  │  Redis       │         │  Manager     │                  │
│  │              │         │              │                  │
│  └──────────────┘         └──────────────┘                  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Components

| Component | Purpose | Scaling |
|-----------|---------|---------|
| **API Service** | Handles HTTP requests, user authentication, business logic | Auto-scales 0-10 instances |
| **Worker Service** | Processes background jobs (emails, payouts, cleanup) | Auto-scales 0-5 instances |
| **Cloud SQL** | PostgreSQL database with pgvector extension | Configurable tier |
| **Cloud Scheduler** | Triggers periodic background jobs | 4 jobs running |
| **Secret Manager** | Stores sensitive configuration (API keys, passwords) | N/A |
| **Memorystore Redis** | Caching and session storage | 5GB memory |

## Prerequisites

### Required Tools

```bash
# 1. Google Cloud SDK
gcloud --version
# If not installed: https://cloud.google.com/sdk/docs/install

# 2. Terraform
terraform --version
# If not installed: https://developer.hashicorp.com/terraform/downloads

# 3. Docker (for local testing)
docker --version

# 4. Git
git --version
```

### Required Permissions

Your GCP account needs these roles:
- `roles/owner` (or combination of):
  - `roles/run.admin`
  - `roles/cloudsql.admin`
  - `roles/secretmanager.admin`
  - `roles/iam.serviceAccountAdmin`
  - `roles/cloudscheduler.admin`

### Initial GCP Setup

```bash
# 1. Set your project
export PROJECT_ID="gen-lang-client-0265949535"
gcloud config set project $PROJECT_ID

# 2. Authenticate
gcloud auth login
gcloud auth application-default login

# 3. Enable required APIs
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  cloudscheduler.googleapis.com \
  redis.googleapis.com \
  artifactregistry.googleapis.com
```

## Initial Setup

### 1. Database Setup

**Create Cloud SQL instance:**

```bash
gcloud sql instances create hauslet-postgres-primary \
  --database-version=POSTGRES_15 \
  --tier=db-custom-2-8192 \
  --region=europe-north1 \
  --network=default \
  --no-assign-ip \
  --database-flags=cloudsql.iam_authentication=on
```

**Create database and user:**

```bash
# Connect to instance
gcloud sql connect hauslet-postgres-primary --user=postgres --quiet

# In PostgreSQL prompt:
CREATE DATABASE hauslet;
CREATE USER hauslet WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE hauslet TO hauslet;

# Enable pgvector extension
\c hauslet
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

\q
```

### 2. Create Secrets

**Store all sensitive configuration in Secret Manager:**

```bash
# Database credentials
echo -n "/cloudsql/$PROJECT_ID:europe-north1:hauslet-postgres-primary" | \
  gcloud secrets create DB_HOST --data-file=-

echo -n "5432" | gcloud secrets create DB_PORT --data-file=-
echo -n "hauslet" | gcloud secrets create DB_NAME --data-file=-
echo -n "hauslet" | gcloud secrets create DB_USER --data-file=-
echo -n "your-db-password" | gcloud secrets create DB_PASSWORD --data-file=-
echo -n "disable" | gcloud secrets create DB_SSLMODE --data-file=-

# Application secrets
echo -n "production" | gcloud secrets create APP_ENV --data-file=-
echo -n "your-jwt-secret" | gcloud secrets create JWT_SECRET --data-file=-
echo -n "your-encryption-key" | gcloud secrets create ENCRYPTION_KEY --data-file=-

# API keys
echo -n "your-anthropic-key" | gcloud secrets create ANTHROPIC_API_KEY --data-file=-
echo -n "your-gemini-key" | gcloud secrets create GEMINI_API_KEY --data-file=-
echo -n "your-resend-key" | gcloud secrets create RESEND_API_KEY --data-file=-

# Payment providers
echo -n "your-paystack-secret" | gcloud secrets create PAYSTACK_SECRET_KEY --data-file=-
echo -n "your-paystack-public" | gcloud secrets create PAYSTACK_PUBLIC_KEY --data-file=-
echo -n "your-stripe-secret" | gcloud secrets create STRIPE_SECRET_KEY --data-file=-
echo -n "your-stripe-publishable" | gcloud secrets create STRIPE_PUBLISHABLE_KEY --data-file=-
echo -n "your-stripe-webhook-secret" | gcloud secrets create STRIPE_WEBHOOK_SECRET --data-file=-

# Storage (Cloudflare R2)
echo -n "your-r2-account-id" | gcloud secrets create R2_ACCOUNT_ID --data-file=-
echo -n "your-r2-access-key" | gcloud secrets create R2_ACCESS_KEY_ID --data-file=-
echo -n "your-r2-secret" | gcloud secrets create R2_ACCESS_KEY_SECRET --data-file=-
echo -n "your-bucket-name" | gcloud secrets create R2_BUCKET_NAME --data-file=-
echo -n "your-r2-endpoint" | gcloud secrets create R2_ENDPOINT --data-file=-
echo -n "your-cdn-host" | gcloud secrets create CDN_HOST --data-file=-

# Redis
echo -n "your-redis-address:6379" | gcloud secrets create REDIS_ADDR --data-file=-

# OAuth
echo -n "your-google-client-id" | gcloud secrets create GOOGLE_CLIENT_ID --data-file=-
echo -n "your-google-client-secret" | gcloud secrets create GOOGLE_CLIENT_SECRET --data-file=-
echo -n "https://your-domain.com/auth/callback" | gcloud secrets create REDIRECT_URL --data-file=-

# Email
echo -n "noreply@your-domain.com" | gcloud secrets create EMAIL_FROM --data-file=-

# Cloud Tasks
echo -n "$PROJECT_ID" | gcloud secrets create CLOUD_TASKS_PROJECT_ID --data-file=-
echo -n "europe-west2" | gcloud secrets create CLOUD_TASKS_LOCATION --data-file=-
echo -n "https://hauslet-worker-staging-xxx.run.app" | gcloud secrets create CLOUD_TASKS_WORKER_URL --data-file=-
echo -n "hauslet-worker-staging-sa@$PROJECT_ID.iam.gserviceaccount.com" | gcloud secrets create CLOUD_TASKS_SERVICE_ACCOUNT --data-file=-

# FX API
echo -n "your-fx-api-key" | gcloud secrets create FX_API_KEY --data-file=-
echo -n "https://v6.exchangerate-api.com/v6" | gcloud secrets create FX_URL --data-file=-
```

### 3. Infrastructure Deployment (Terraform)

**Deploy base infrastructure:**

```bash
cd deploy/terraform

# Initialize Terraform
terraform init

# Review what will be created
terraform plan

# Apply configuration
terraform apply

# You'll see outputs:
# - Service account emails
# - Artifact registry URL
# - Scheduler job details
```

## Staging Deployment

### Quick Deployment

```bash
cd deploy
./deploy-staging.sh
```

This script will:
1. ✅ Verify VPC connector exists (creates if missing)
2. ✅ Build Docker images for API and Worker
3. ✅ Push images to Artifact Registry
4. ✅ Deploy to Cloud Run (staging services)
5. ✅ Run smoke tests

### Manual Deployment

```bash
# Set variables
export PROJECT_ID="gen-lang-client-0265949535"
export REGION="europe-north1"
export TAG=$(git rev-parse --short HEAD)

# Build and deploy
gcloud builds submit \
  --config=cloudbuild-staging.yaml \
  --region=$REGION \
  --substitutions=_TAG=$TAG
```

### Verify Deployment

```bash
# Check API health
API_URL=$(gcloud run services describe hauslet-api-staging \
  --region=europe-north1 --format='value(status.url)')

curl $API_URL/health
# Expected: {"status":"ok"}

# Check worker logs
gcloud run services logs read hauslet-worker-staging \
  --region=europe-north1 --limit=20

# Check scheduler jobs
gcloud scheduler jobs list --location=europe-west1
```

## Production Deployment

### Prerequisites

1. ✅ Staging tested and verified
2. ✅ Database migrations tested
3. ✅ Secrets configured for production
4. ✅ DNS configured (if using custom domain)

### Deployment Steps

```bash
# 1. Update Terraform for production
cd deploy/terraform
# Edit terraform.tfvars to use production values

# 2. Deploy production infrastructure
terraform apply

# 3. Deploy services
cd ..
./deploy-production.sh  # Create this similar to deploy-staging.sh

# Or manual:
gcloud builds submit \
  --config=cloudbuild.yaml \
  --region=europe-north1 \
  --substitutions=_TAG=$(git rev-parse --short HEAD)
```

### Production Checklist

- [ ] Database backup configured
- [ ] Monitoring and alerting set up
- [ ] Custom domain configured
- [ ] SSL certificates issued
- [ ] Rate limiting enabled
- [ ] CORS properly configured
- [ ] Secrets rotated from staging
- [ ] Error tracking enabled (Sentry, etc.)
- [ ] Load testing completed

## Infrastructure Management

### Updating Services

**Code changes only:**
```bash
cd deploy
./deploy-staging.sh  # Or deploy-production.sh
```

**Infrastructure changes (Terraform):**
```bash
cd deploy/terraform

# Edit .tf files
vim cloudscheduler.tf  # For example

# Review changes
terraform plan

# Apply changes
terraform apply
```

### Adding a New Scheduler Job

1. Edit `deploy/terraform/cloudscheduler.tf`:

```hcl
resource "google_cloud_scheduler_job" "new_job" {
  name        = "new-job-scheduler"
  description = "Description of what this job does"
  schedule    = "0 9 * * *"  # Daily at 9am
  time_zone   = "UTC"
  region      = "europe-west1"

  retry_config {
    retry_count = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
  }

  http_target {
    uri         = "${var.worker_url}/tasks/your/endpoint"
    http_method = "POST"

    headers = {
      "Content-Type" = "application/json"
    }

    body = base64encode(jsonencode({
      param1 = "value1"
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
```

2. Apply changes:
```bash
terraform apply
```

3. Create handler in code (if new endpoint)

4. Test:
```bash
gcloud scheduler jobs run new-job-scheduler --location=europe-west1
```

### Scaling Services

**Manual scaling:**
```bash
# Update Cloud Run service
gcloud run services update hauslet-api-staging \
  --region=europe-north1 \
  --min-instances=1 \
  --max-instances=20
```

**Auto-scaling (default):**
- Scales to 0 when no traffic
- Scales up based on CPU and request metrics
- Configure in `cloudbuild-staging.yaml`

### Database Migrations

Migrations run automatically when API starts:
- API service checks database schema on startup
- Runs pending migrations via GORM AutoMigrate
- Worker does NOT run migrations (read-only)

**Manual migration check:**
```bash
# Check current schema
gcloud sql connect hauslet-postgres-primary --user=hauslet --database=hauslet

# In psql:
\dt  # List tables
\d+ users  # Describe specific table
```

## Troubleshooting

### Service Won't Start

**Check logs:**
```bash
gcloud run services logs read hauslet-api-staging \
  --region=europe-north1 \
  --limit=50
```

**Common issues:**

1. **Permission denied on config files**
   - Symptom: `open config/defaults/calendar.yaml: permission denied`
   - Fix: Ensure Dockerfiles use `--chown=nonroot:nonroot`

2. **Database connection refused**
   - Symptom: `dial unix /cloudsql/...: connect: connection refused`
   - Fix: Service account missing `roles/cloudsql.client`
   ```bash
   gcloud projects add-iam-policy-binding $PROJECT_ID \
     --member="serviceAccount:hauslet-api-staging-sa@$PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/cloudsql.client"
   ```

3. **Secrets not found**
   - Symptom: `ENV JWT_SECRET is missing`
   - Fix: Check secret exists and service account has access
   ```bash
   gcloud secrets describe JWT_SECRET
   gcloud secrets add-iam-policy-binding JWT_SECRET \
     --member="serviceAccount:hauslet-api-staging-sa@$PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/secretmanager.secretAccessor"
   ```

4. **Vector type not found**
   - Symptom: `ERROR: type "vector" does not exist`
   - Fix: Enable pgvector in database
   ```bash
   gcloud sql connect hauslet-postgres-primary --user=postgres --database=hauslet
   CREATE EXTENSION IF NOT EXISTS vector;
   ```

### Scheduler Jobs Not Running

**Check job status:**
```bash
gcloud scheduler jobs describe media-cleanup-scheduler \
  --location=europe-west1
```

**Check logs:**
```bash
# Scheduler execution logs
gcloud logging read "resource.type=cloud_scheduler_job" \
  --limit=20 --format=json

# Worker logs for job processing
gcloud run services logs read hauslet-worker-staging \
  --region=europe-north1 --limit=50
```

**Common issues:**

1. **403 Forbidden**
   - Fix: Grant invoker permission
   ```bash
   gcloud run services add-iam-policy-binding hauslet-worker-staging \
     --region=europe-north1 \
     --member="serviceAccount:hauslet-worker-staging-sa@$PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/run.invoker"
   ```

2. **404 Not Found**
   - Check endpoint exists in worker code
   - Verify URL in scheduler job config

### Performance Issues

**Check metrics:**
```bash
# Open monitoring dashboard
open "https://console.cloud.google.com/run/detail/europe-north1/hauslet-api-staging/metrics?project=$PROJECT_ID"
```

**Scale up if needed:**
```bash
gcloud run services update hauslet-api-staging \
  --region=europe-north1 \
  --min-instances=2 \
  --max-instances=50 \
  --cpu=4 \
  --memory=4Gi
```

### Cost Optimization

**View current costs:**
```bash
# Open billing dashboard
open "https://console.cloud.google.com/billing?project=$PROJECT_ID"
```

**Reduce costs:**

1. **Scale to zero when idle:**
   ```bash
   gcloud run services update SERVICE_NAME \
     --min-instances=0
   ```

2. **Use smaller instance types:**
   ```bash
   gcloud run services update SERVICE_NAME \
     --cpu=1 \
     --memory=512Mi
   ```

3. **Delete unused resources:**
   ```bash
   # List all services
   gcloud run services list

   # Delete unused service
   gcloud run services delete OLD_SERVICE --region=europe-north1
   ```

## Monitoring & Alerts

### Set Up Error Alerts

```bash
# Create notification channel (email)
gcloud alpha monitoring channels create \
  --display-name="Team Email" \
  --type=email \
  --channel-labels=email_address=team@your-domain.com

# Create alert policy for 500 errors
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="API 500 Errors" \
  --condition-display-name="High Error Rate" \
  --condition-threshold-value=10 \
  --condition-threshold-duration=300s
```

### Logging

**View structured logs:**
```bash
# Recent errors
gcloud logging read "severity>=ERROR" \
  --limit=50 \
  --format=json

# Specific service
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-api-staging" \
  --limit=50
```

## Backup & Recovery

### Database Backups

**Enable automated backups:**
```bash
gcloud sql instances patch hauslet-postgres-primary \
  --backup-start-time=03:00 \
  --enable-bin-log
```

**Manual backup:**
```bash
gcloud sql backups create \
  --instance=hauslet-postgres-primary \
  --description="Pre-deployment backup"
```

**Restore from backup:**
```bash
# List backups
gcloud sql backups list --instance=hauslet-postgres-primary

# Restore
gcloud sql backups restore BACKUP_ID \
  --backup-instance=hauslet-postgres-primary \
  --backup-id=BACKUP_ID
```

## Security Best Practices

1. **Rotate secrets regularly** (every 90 days)
2. **Use least privilege IAM** (don't use `roles/owner`)
3. **Enable VPC Service Controls** for production
4. **Use Cloud Armor** for DDoS protection
5. **Enable audit logging**
6. **Keep dependencies updated**
7. **Use Secret Manager**, never commit secrets to Git

## Useful Links

- [GCP Console](https://console.cloud.google.com)
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)
- [Terraform GCP Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [Project Repository](https://github.com/your-org/hauslet-services)

## Support

For issues or questions:
1. Check logs: `gcloud run services logs read SERVICE_NAME`
2. Review this guide
3. Check [Troubleshooting](#troubleshooting) section
4. Contact DevOps team
