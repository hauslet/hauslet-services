# GCP Deployment Migration Plan

## ⚠️ Critical Warnings

Before proceeding with deployment, address these critical issues:

### 1. Database Migration Race Condition 🚨
**Problem**: `cmd/api/main.go` runs `database.RunMigrations` on startup. Multiple Cloud Run instances starting simultaneously can race the migration step.

**Solution**: Keep `database.RunMigrations` as the migration path and make it safe for concurrent Cloud Run startups by using a database-level advisory lock (only one instance migrates at a time).

**Impact**: HIGH - Can corrupt database schema in production

---

### 2. Job Timeout Configuration 🚨
**Problem**: Worker processor has a hard 15-second timeout. Tasks like AI moderation (video processing in `gemini.go`) and media thumbnail generation can exceed this limit.

**Solution**:
- Increase Cloud Run request timeout to 300s (5 minutes) for Worker service
- Configure per-queue timeouts in Cloud Tasks (ai_moderation: 300s, media_thumbnail: 180s)
- Remove hard 15s context timeout from worker HTTP handlers

**Impact**: HIGH - Long-running tasks will fail and retry infinitely

---

### 3. Queue Fallback Behavior in Production ⚠️
**Problem**: Queue publishers use best-effort fallbacks (queue fails → send directly). This masks infrastructure issues in production.

**Solution**: Implement environment-aware behavior:
- **Development/Staging**: Keep fallback for convenience
- **Production**: Fail-fast to alert on infrastructure problems

**Impact**: MEDIUM - Masks queue infrastructure issues, harder to debug failures

---

## Executive Summary

Migrate Hauslet Services from local/Docker development to Google Cloud Platform production deployment with:
- **Cloud Run** for API and Worker services (serverless, auto-scaling)
- **Cloud Tasks** replacing NATS JetStream (9 active queues)
- **Cloud Scheduler** replacing 4 manual tickers
- **Cloud SQL for PostgreSQL** (recommended over AlloyDB for cost/performance balance)
- **Valkey (Redis-compatible)** (sessions + caching)
- **Cloud Build + Artifact Registry** for CI/CD
- **Secret Manager** for credentials
- Keep **Cloudflare R2** for object storage (no migration needed)

---

## Architecture Overview

### Current Architecture
```
┌─────────────┐     ┌──────────────┐
│   API       │────▶│ PostgreSQL   │
│   Server    │     │ (local)      │
└──────┬──────┘     └──────────────┘
       │
       │ Publish    ┌──────────────┐
       ├───────────▶│ NATS         │
       │            │ JetStream    │
       │            └──────┬───────┘
       │                   │ Consume
       │            ┌──────▼───────┐
       │            │   Worker     │
       │            │   Process    │
       │            └──────┬───────┘
       │                   │
       └─────────┬─────────┘
                 │
         ┌───────▼────────┐
         │ Redis (local)  │
         │ Sessions/Cache │
         └────────────────┘
```

### Target GCP Architecture
```
                    ┌──────────────────┐
                    │ Cloud Load       │
                    │ Balancer (HTTPS) │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │ Cloud Run        │
                    │ (API Service)    │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
      ┌───────▼──────┐ ┌────▼─────┐ ┌──────▼────────┐
      │ Cloud SQL    │ │  Valkey  │ │ Cloud Tasks   │
      │ PostgreSQL   │ │ (cache)  │ │ (9 queues)    │
      └──────────────┘ └───────────┘ └──────┬────────┘
                                             │ HTTP POST
                                      ┌──────▼─────────┐
                                      │ Cloud Run      │
                                      │ (Worker Service)│
                                      └────────────────┘

      ┌──────────────────────────────────────┐
      │ Cloud Scheduler (4 cron jobs)        │
      │ → Posts to Cloud Tasks               │
      └──────────────────────────────────────┘

      ┌──────────────────────────────────────┐
      │ Cloud Build (CI/CD)                  │
      │ → Artifact Registry                  │
      └──────────────────────────────────────┘
```

---

## Migration Phases

### Phase 1: Infrastructure Setup (Week 1)
1. GCP project setup and IAM
2. Enable required APIs
3. Provision Cloud SQL PostgreSQL
4. Provision Valkey (managed cache)
5. Set up Secret Manager
6. Configure VPC and networking

### Phase 2: Containerization (Week 1-2)
1. Create Dockerfiles for API and Worker
2. Add health check endpoints
3. Update configuration for Cloud Run
4. Local Docker testing
5. Multi-stage builds for optimization

### Phase 3: Cloud Tasks Migration (Week 2-3)
1. Create Cloud Tasks queues (9 active queues)
2. Implement Cloud Tasks adapter (replaces NATS client)
3. Create HTTP task handlers in Worker
4. Update all queue publishers with environment-aware fallback
5. Add retry policies and dead-letter queues

### Phase 4: Cloud Scheduler Migration (Week 3)
1. Create 4 Cloud Scheduler jobs
2. Replace manual tickers with HTTP endpoints
3. Configure cron schedules
4. Test scheduler → Cloud Tasks flow

### Phase 5: CI/CD Pipeline (Week 3-4)
1. Set up Cloud Build triggers
2. Create build configurations (cloudbuild.yaml)
3. Configure Artifact Registry
4. Implement deployment pipeline
5. Set up staging and production environments

### Phase 6: Deployment & Testing (Week 4)
1. Deploy to staging environment
2. Integration testing
3. Performance testing
4. Security audit
5. Production deployment

### Phase 7: Monitoring & Optimization (Week 5)
1. Set up Cloud Monitoring
2. Configure alerting
3. Implement Cloud Trace
4. Cost optimization
5. Documentation

---

## Database Recommendation: Cloud SQL vs AlloyDB

**Recommendation: Cloud SQL for PostgreSQL** ✅

**Reasoning:**
- **Cost-effective**: ~70% cheaper than AlloyDB for similar configurations
- **Sufficient performance**: Up to 64 vCPUs, 416GB RAM, 30,000 IOPS
- **Proven track record**: Mature service with excellent reliability
- **Easy migration**: Standard PostgreSQL compatibility, familiar tooling
- **Your workload**: Booking/property platform doesn't need AlloyDB's extreme performance initially

**When to consider AlloyDB later:**
- Scale to 10,000+ concurrent bookings/second
- Need sub-millisecond query latency at extreme scale
- ML-powered query optimization becomes critical
- Budget allows for 3-4x database costs

Start with **Cloud SQL** and migrate to AlloyDB later if needed (minimal code changes required).

---

## NATS → Cloud Tasks Migration

### Queue Mapping

**Active Queues (9 total)**:

| NATS Subject | Cloud Tasks Queue | Handler Endpoint | Rate Limit | Timeout |
|-------------|-------------------|------------------|------------|---------|
| `email.send` | `email-queue` | `/tasks/email` | 100/s | 30s |
| `media.thumbnail` | `media-thumbnail-queue` | `/tasks/media/thumbnail` | 20/s | 180s |
| `media.cleanup` | `media-cleanup-queue` | `/tasks/media/cleanup` | 5/s | 60s |
| `ai.moderation` | `ai-moderation-queue` | `/tasks/ai/moderation` | 10/s | 300s |
| `booking.expiry.check` | `booking-expiry-queue` | `/tasks/booking/expiry` | 10/s | 60s |
| `booking.refund` | `booking-refund-queue` | `/tasks/booking/refund` | 20/s | 60s |
| `payment.webhook` | `payment-webhook-queue` | `/tasks/payment/webhook` | 50/s | 60s |
| `payout.process` | `payout-process-queue` | `/tasks/finance/payout/process` | 5/s | 120s |
| `payout.retry` | `payout-retry-queue` | `/tasks/finance/payout/retry` | 10/s | 120s |

**Note**: `notification.push` and `moderation.check` are defined in `config/defaults/queue.yaml` but have no active publishers in the codebase. These have been excluded from the migration.

### Additional Required Updates

- **Task authentication**: Configure Cloud Tasks to attach an OIDC token and verify it in Worker task handlers. Lock task endpoints to Cloud Tasks service account only.
- **Payload size**: Current `email.send` jobs include full HTML bodies. Cloud Tasks has a 100KB payload limit, so move email rendering to the worker (template + data) or store HTML in object storage and pass a reference.

### Code Changes Required

**New Files to Create:**

1. `internal/platform/cloudtasks/client.go` - Cloud Tasks adapter with environment-aware fallback
2. `internal/platform/cloudtasks/config.go` - Queue configurations (9 queues)
3. `cmd/worker/server.go` - HTTP server for Worker with 9 task endpoints
4. `internal/transport/http/health/handler.go` - Health check endpoints
5. `internal/platform/secrets/client.go` - Secret Manager client
6. `deploy/Dockerfile.api` - API service container image
7. `deploy/Dockerfile.worker` - Worker service container image (300s timeout)
8. `cloudbuild.yaml` - CI/CD pipeline (build/deploy only)
9. `deploy/terraform/main.tf` - Complete infrastructure as code

**Files to Modify:**

1. `internal/modules/*/notification/service.go` (7 files) - Replace NATS with Cloud Tasks
2. `internal/modules/property/service/listing_media.go` - Replace NATS with Cloud Tasks
3. `internal/modules/moderation/service/service.go` - Replace NATS with Cloud Tasks
4. `internal/modules/payments/port/http/webhook_handler.go` - Replace NATS with Cloud Tasks
5. `cmd/worker/main.go` - Remove NATS, add HTTP server, remove tickers
6. **`cmd/api/main.go` - Add health routes, Cloud SQL config, keep `RunMigrations`**
7. `internal/platform/database/migrate.go` - Add advisory lock around migrations
8. `internal/platform/database/connection.go` - Cloud SQL Unix socket support
9. `config/structs.go` / `config/config.go` - Add `DB_INSTANCE_CONNECTION_NAME` config
10. `internal/platform/redis/redis.go` - Valkey connection configuration (VPC/private)

**Files to Delete:**

1. `internal/transport/worker/processor.go` - NATS consumer (no longer needed)
2. `config/defaults/queue.yaml` - Update values to Cloud Tasks queue names (keep file)
3. `cmd/worker/setup/handlers.go` - Lines 349-460 (all 4 ticker functions)

---

## Manual Tickers → Cloud Scheduler Migration

### Scheduler Jobs to Create

| Current Ticker | Cloud Scheduler Job | Schedule | Target Queue |
|---------------|---------------------|----------|--------------|
| `StartPeriodicCleanup()` | `media-cleanup-scheduler` | `*/15 * * * *` | `media-cleanup-queue` |
| `StartBookingExpiryCheck()` | `booking-expiry-scheduler` | `*/2 * * * *` | `booking-expiry-queue` |
| `StartPayoutProcessing()` | `payout-process-scheduler` | `0 * * * *` | `payout-process-queue` |
| `StartDisbursementRetry()` | `disbursement-retry-scheduler` | `*/15 * * * *` | `payout-retry-queue` |

**Schedule Details:**
- Media cleanup: Every 15 minutes
- Booking expiry: Every 2 minutes (configurable, PRD suggests 5)
- Payout processing: Every hour on the hour (configurable via platform.yaml)
- Disbursement retry: Every 15 minutes (configurable via platform.yaml)

---

## Database Migration Strategy

### Problem: Auto-Migrate Race Condition

**Current Code** (`cmd/api/main.go` runs `database.RunMigrations` on startup):
```go
if err := database.RunMigrations(db, log,
    &schema.User{},
    &schema.Property{},
    &schema.Booking{},
    // ... 20+ models
); err != nil {
    log.Fatalf("failed to run migrations: %v", err)
}
```

**Issue**: Cloud Run can start multiple instances concurrently. Each instance would attempt `RunMigrations` at the same time.

### Solution: Make `RunMigrations` Safe with an Advisory Lock

**Keep `database.RunMigrations` as the primary migration path**, but wrap it with a DB advisory lock so only one instance migrates at a time. Other instances should block briefly, then continue once the lock is released.

**Planned change**:

- `internal/platform/database/migrate.go`: acquire `pg_advisory_lock` before AutoMigrate; release after.
- `cmd/api/main.go`: keep calling `RunMigrations` as-is.

**Note on raw SQL**: For one-off schema changes, run manual SQL (or dedicated scripts) directly against Cloud SQL as needed. This avoids introducing a second migration system.

### Benefits

✅ **No race conditions** - Advisory lock serializes concurrent `RunMigrations`
✅ **Single migration path** - `RunMigrations` remains the standard flow
✅ **Local dev unchanged** - `RunMigrations` still uses AutoMigrate for development
✅ **Manual control** - Raw SQL can be applied when needed

---

## Job Timeout Configuration

### Problem: Hard 15-Second Timeout

**Current Code** (`internal/transport/worker/processor.go`):
```go
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
```

**Tasks that exceed 15s**:
- AI moderation with video upload/polling (`internal/platform/ai/gemini.go`)
- Media thumbnail generation for large images
- Payout batch processing
- Complex booking refunds

### Solution: Per-Queue Timeout Configuration

**1. Update Cloud Run Worker Service** - Increase request timeout to 300s (5 minutes):

```yaml
# terraform/cloudrun.tf
resource "google_cloud_run_service" "worker" {
  # ... existing config ...

  template {
    spec {
      timeout_seconds = 300  # 5 minutes (was 60s default)

      containers {
        # ... existing config ...
      }
    }
  }
}
```

**2. Configure Cloud Tasks Queue Timeouts**:

```yaml
# terraform/cloudtasks.tf
resource "google_cloud_tasks_queue" "ai_moderation" {
  name     = "ai-moderation-queue"
  location = var.region

  retry_config {
    max_attempts       = 3
    max_retry_duration = "900s"  # 15 minutes total
    min_backoff        = "10s"
    max_backoff        = "300s"
  }

  # Task execution timeout (per attempt)
  task_ttl = "300s"  # 5 minutes
}

resource "google_cloud_tasks_queue" "media_thumbnail" {
  name     = "media-thumbnail-queue"
  location = var.region

  retry_config {
    max_attempts = 3
    max_retry_duration = "540s"  # 9 minutes
  }

  task_ttl = "180s"  # 3 minutes per attempt
}

# Shorter timeouts for fast tasks
resource "google_cloud_tasks_queue" "email" {
  name     = "email-queue"
  location = var.region

  retry_config {
    max_attempts = 5
    max_retry_duration = "600s"
  }

  task_ttl = "30s"  # Email should be quick
}
```

**3. Remove Hard Timeout from Worker Handlers**:

```go
// cmd/worker/server.go - HTTP handlers
func (s *Server) handleAIModerationTask(w http.ResponseWriter, r *http.Request) {
    // Use request context (inherits Cloud Run timeout)
    ctx := r.Context()  // No 15s timeout!

    var job jobs.AIModerationJob
    if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
        http.Error(w, "invalid job payload", http.StatusBadRequest)
        return
    }

    // Process task (can take up to 300s)
    if err := s.aiModerationHandler.Handle(ctx, job); err != nil {
        s.log.Logf("ERROR AI moderation failed: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

### Timeout Summary

| Queue | Task Timeout | Max Retries | Total Duration |
|-------|-------------|-------------|----------------|
| ai-moderation | 300s (5 min) | 3 | 15 min |
| media-thumbnail | 180s (3 min) | 3 | 9 min |
| email | 30s | 5 | 2.5 min |
| payment-webhook | 60s | 5 | 5 min |
| payout-process | 120s (2 min) | 3 | 6 min |
| booking-expiry | 60s | 3 | 3 min |

---

## Environment-Aware Queue Fallback Pattern

### Problem: Best-Effort Fallbacks Mask Infrastructure Issues

**Current Pattern** (used in 7+ notification services):
```go
// Always falls back to direct send if queue fails
if err := s.queueClient.Publish(ctx, "email.send", job); err != nil {
    s.log.Logf("WARN queue publish failed, sending directly: %v", err)
    return s.emailService.SendDirectly(ctx, job)  // Masks queue outage!
}
```

**Issue**: In production, queue failures should alert ops team, not silently fallback.

### Solution: Environment-Aware Behavior

```go
// internal/platform/cloudtasks/client.go
type Client struct {
    projectID string
    location  string
    client    *cloudtasks.Client
    log       *lgr.Logger

    // Environment-aware behavior
    environment string  // "development", "staging", "production"
}

func (c *Client) Publish(ctx context.Context, queueName string, payload interface{}) error {
    // ... create task ...

    _, err := c.client.CreateTask(ctx, req)
    if err != nil {
        c.log.Logf("ERROR failed to publish to queue %s: %v", queueName, err)

        // Environment-specific handling
        if c.environment == "production" {
            // Fail-fast in production - force alerting
            return fmt.Errorf("queue publish failed (production): %w", err)
        }

        // Allow graceful degradation in dev/staging
        c.log.Logf("WARN queue unavailable, caller may fallback (env=%s)", c.environment)
        return err
    }

    return nil
}
```

**Updated Service Pattern**:

```go
// internal/modules/booking/notification/service.go
func (s *NotificationService) NotifyBookingConfirmed(ctx context.Context, booking *domain.Booking) error {
    job := jobs.EmailJob{
        Type:      "booking_confirmed",
        BookingID: booking.ID,
        // ... other fields ...
    }

    err := s.queueClient.Publish(ctx, "email-queue", job)
    if err != nil {
        // Check if we're in production
        if os.Getenv("APP_ENV") == "production" {
            // Don't fallback - return error and let it bubble up
            // Ops will be alerted via error rate monitoring
            return fmt.Errorf("failed to queue booking confirmation email: %w", err)
        }

        // Dev/staging: Allow graceful fallback
        s.log.Logf("WARN queue unavailable, sending email directly (dev mode)")
        return s.emailService.SendBookingConfirmed(ctx, booking)
    }

    return nil
}
```

**Configuration** (via environment variable):

```bash
# Local development
APP_ENV=development

# Staging
APP_ENV=staging

# Production (Cloud Run)
APP_ENV=production
```

### Behavior Matrix

| Environment | Queue Fails | Behavior |
|-------------|------------|----------|
| **development** | ✅ Yes | Fallback to direct send, log warning |
| **staging** | ✅ Yes | Fallback to direct send, log warning |
| **production** | ❌ No | Return error, trigger alerts, NO fallback |

---

## Cloud SQL Connection Configuration

### Unix Socket Connection for Cloud Run

**Current Code** (`internal/platform/database/connection.go`):
```go
// Traditional host:port connection
dsn := fmt.Sprintf(
    "host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
    cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DbName, cfg.DBPort, cfg.DBSslmode,
)
```

**Updated for Cloud Run**:

```go
// internal/platform/database/connection.go
func ConnectPostgreSQL(cfg config.DBConfig) (*gorm.DB, error) {
    var dsn string

    // Cloud SQL connection via Unix socket (Cloud Run)
    if cfg.InstanceConnectionName != "" {
        dsn = fmt.Sprintf(
            "host=/cloudsql/%s user=%s password=%s dbname=%s sslmode=disable",
            cfg.InstanceConnectionName,  // Format: project:region:instance
            cfg.DBUser,
            cfg.DBPassword,
            cfg.DbName,
        )
        log.Printf("Using Cloud SQL Unix socket: %s", cfg.InstanceConnectionName)
    } else {
        // Traditional connection (local dev, Cloud SQL Proxy)
        dsn = fmt.Sprintf(
            "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
            cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DbName, cfg.DBSslmode,
        )
        log.Printf("Using traditional PostgreSQL connection: %s:%s", cfg.DBHost, cfg.DBPort)
    }

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    sqlDB.SetMaxOpenConns(25)      // Cloud SQL default
    sqlDB.SetMaxIdleConns(5)
    sqlDB.SetConnMaxLifetime(30 * time.Minute)

    return db, nil
}
```

**Environment Configuration**:

```bash
# Local development
DATABASE_HOST=localhost
DATABASE_PORT=5432

# Cloud Run (production)
DB_INSTANCE_CONNECTION_NAME=hauslet-prod:us-central1:hauslet-postgres
DB_USER=hauslet
DB_NAME=hauslet
# DB_PASSWORD from Secret Manager
```

**Cloud Run Service Configuration** (Terraform):

```hcl
resource "google_cloud_run_service" "api" {
  # ... existing config ...

  template {
    metadata {
      annotations = {
        # Enable Cloud SQL connection
        "run.googleapis.com/cloudsql-instances" = google_sql_database_instance.postgres.connection_name
      }
    }

    spec {
      containers {
        env {
          name  = "INSTANCE_CONNECTION_NAME"
          value = google_sql_database_instance.postgres.connection_name
        }
        # ... other env vars ...
      }
    }
  }
}
```

---

## Valkey (Redis-Compatible) Configuration

### VPC Connector Requirement

**Problem**: Managed Valkey typically runs in a VPC/private network. Cloud Run (serverless) needs a VPC connector (or private service access) to reach it.

**Current Code** (`internal/platform/redis/redis.go`):
```go
// Simple connection - works for local dev
client := redis.NewClient(&redis.Options{
    Addr: cfg.Addr,  // e.g. localhost:6379 or redis://user:pass@host:6379
})
```

**Updated for Valkey**:

Keep the existing `REDIS_ADDR` path and point it to the Valkey internal endpoint (VPC-only). No structural code changes are required beyond ensuring network reachability.

**Terraform Configuration**:

```hcl
# VPC for Valkey
resource "google_compute_network" "vpc" {
  name                    = "hauslet-vpc"
  auto_create_subnetworks = true
}

# Managed Valkey (or Memorystore-compatible cache)
resource "google_redis_instance" "cache" {
  name               = "hauslet-redis"
  tier               = "STANDARD_HA"
  memory_size_gb     = 5
  region             = var.region
  authorized_network = google_compute_network.vpc.id
  redis_version      = "REDIS_7_0"

  # VPC-native, no AUTH
  auth_enabled = false
}

# Serverless VPC Access Connector (for Cloud Run)
resource "google_vpc_access_connector" "connector" {
  name          = "hauslet-vpc-connector"
  region        = var.region
  network       = google_compute_network.vpc.name
  ip_cidr_range = "10.8.0.0/28"  # Small range for connector

  # Minimum throughput
  min_throughput = 200  # Mbps
  max_throughput = 300
}

# Output Redis IP for configuration
output "redis_host" {
  value       = "${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
  description = "Valkey host:port"
}
```

**Cloud Run Service Configuration**:

```hcl
resource "google_cloud_run_service" "api" {
  # ... existing config ...

  template {
    metadata {
      annotations = {
        # Attach VPC connector to access Valkey
        "run.googleapis.com/vpc-access-connector" = google_vpc_access_connector.connector.name
        "run.googleapis.com/vpc-access-egress"    = "private-ranges-only"
      }
    }

    spec {
      containers {
        env {
          name = "REDIS_HOST"
          value_source {
            secret_key_ref {
              name = "REDIS_HOST"
              key  = "latest"
            }
          }
        }
        # ... other env vars ...
      }
    }
  }
}
```

**Environment Variables**:

```bash
# Local development
REDIS_HOST=localhost
REDIS_PORT=6379

# Cloud Run (production)
REDIS_ADDR=10.0.0.3:6379  # Valkey internal endpoint (from Terraform output)
```

**Important Notes**:
- Valkey HA tier recommended for production
- AUTH is disabled for VPC-native access (no password needed)
- VPC connector adds ~$8/month cost
- Redis is NOT accessible from internet (VPC-only)

---

## Deployment Guide

### Prerequisites

1. GCP Project created
2. Billing enabled
3. gcloud CLI installed and authenticated
4. Terraform installed (v1.5+)
5. GitHub repository access

### Step 1: GCP Project Setup

```bash
# Set project
export PROJECT_ID="hauslet-prod"
export REGION="us-central1"

gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  cloudtasks.googleapis.com \
  cloudscheduler.googleapis.com \
  secretmanager.googleapis.com \
  vpcaccess.googleapis.com \
  artifactregistry.googleapis.com \
  compute.googleapis.com
```

### Step 2: Terraform Infrastructure Provisioning

```bash
cd deploy/terraform

# Initialize Terraform
terraform init

# Review plan
terraform plan \
  -var="project_id=$PROJECT_ID" \
  -var="region=$REGION" \
  -out=tfplan

# Apply (creates all infrastructure)
terraform apply tfplan
```

**Resources Created:**
- Cloud SQL PostgreSQL instance (~10 minutes)
- Valkey cache (~5 minutes)
- VPC and Serverless VPC Connector
- 9 Cloud Tasks queues
- 4 Cloud Scheduler jobs
- Service accounts and IAM bindings
- Cloud Run services (initial deployment)

### Step 3: Secret Manager Setup

```bash
# Create secrets from .env file or manually
echo -n "your-db-password" | gcloud secrets create DB_PASSWORD --data-file=-
echo -n "your-jwt-secret" | gcloud secrets create JWT_SECRET --data-file=-
echo -n "your-paystack-key" | gcloud secrets create PAYSTACK_SECRET_KEY --data-file=-
echo -n "your-stripe-key" | gcloud secrets create STRIPE_SECRET_KEY --data-file=-
echo -n "your-gemini-key" | gcloud secrets create GEMINI_API_KEY --data-file=-
echo -n "your-anthropic-key" | gcloud secrets create ANTHROPIC_API_KEY --data-file=-
echo -n "your-resend-key" | gcloud secrets create RESEND_API_KEY --data-file=-
echo -n "your-r2-secret" | gcloud secrets create R2_SECRET_ACCESS_KEY --data-file=-
echo -n "your-google-oauth-secret" | gcloud secrets create OAUTH_GOOGLE_CLIENT_SECRET --data-file=-
echo -n "your-encryption-key" | gcloud secrets create ENCRYPTION_KEY --data-file=-

# Get Valkey endpoint
export REDIS_HOST=$(terraform output -raw redis_host)
echo -n "$REDIS_HOST" | gcloud secrets create REDIS_HOST --data-file=-

# Grant service accounts access to secrets
for SECRET in DB_PASSWORD JWT_SECRET PAYSTACK_SECRET_KEY STRIPE_SECRET_KEY \
  GEMINI_API_KEY ANTHROPIC_API_KEY RESEND_API_KEY R2_SECRET_ACCESS_KEY \
  OAUTH_GOOGLE_CLIENT_SECRET ENCRYPTION_KEY REDIS_HOST; do

  gcloud secrets add-iam-policy-binding $SECRET \
    --member="serviceAccount:hauslet-api-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"

  gcloud secrets add-iam-policy-binding $SECRET \
    --member="serviceAccount:hauslet-worker-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"
done
```

### Step 4: Database Migration

Migrations run via `database.RunMigrations` on API startup (with advisory lock to prevent races).
Use manual SQL only when needed.

```bash
# Get Cloud SQL connection details
export INSTANCE_CONNECTION_NAME=$(terraform output -raw db_instance_connection_name)

# Install Cloud SQL Proxy
wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
chmod +x cloud_sql_proxy

# Start proxy (in background)
./cloud_sql_proxy -instances=$INSTANCE_CONNECTION_NAME=tcp:5432 &
PROXY_PID=$!

# Wait for proxy to start
sleep 3

# Run manual SQL when needed
export DATABASE_URL="postgres://hauslet:$(gcloud secrets versions access latest --secret=DB_PASSWORD)@localhost:5432/hauslet?sslmode=disable"
psql "$DATABASE_URL" -f db/migrations/your_change.sql

# Stop proxy
kill $PROXY_PID
```

### Step 5: Cloud Build Setup

```bash
# Create Artifact Registry repository
gcloud artifacts repositories create hauslet \
  --repository-format=docker \
  --location=$REGION \
  --description="Hauslet Docker images"

# Connect GitHub repository (interactive)
gcloud builds connections create github hauslet-github \
  --region=$REGION

# Create build triggers
gcloud builds triggers create github \
  --name="hauslet-prod-deploy" \
  --repo-name="hauslet-services" \
  --repo-owner="YOUR_GITHUB_ORG" \
  --branch-pattern="^main$" \
  --build-config="cloudbuild.yaml" \
  --region=$REGION

gcloud builds triggers create github \
  --name="hauslet-staging-deploy" \
  --repo-name="hauslet-services" \
  --repo-owner="YOUR_GITHUB_ORG" \
  --branch-pattern="^staging$" \
  --build-config="cloudbuild-staging.yaml" \
  --region=$REGION
```

### Step 6: Initial Deployment

```bash
# Build and deploy manually for first time
gcloud builds submit \
  --config=cloudbuild.yaml \
  --substitutions=_ENV=production \
  --region=$REGION

# Monitor build progress
gcloud builds log --stream

# Verify deployment
gcloud run services describe hauslet-api --region=$REGION
gcloud run services describe hauslet-worker --region=$REGION

# Get service URLs
export API_URL=$(gcloud run services describe hauslet-api --region=$REGION --format='value(status.url)')
export WORKER_URL=$(gcloud run services describe hauslet-worker --region=$REGION --format='value(status.url)')

echo "API URL: $API_URL"
echo "Worker URL: $WORKER_URL"
```

### Step 7: Test Deployment

```bash
# Health checks
curl $API_URL/health
curl $API_URL/health/ready

# API endpoint test
curl $API_URL/api/v1/properties

# Worker health (requires authentication - use Cloud Tasks service account)
# This will be called by Cloud Tasks, not directly
```

### Step 8: Configure Custom Domain (Optional)

```bash
# Map custom domain to Cloud Run API
gcloud run domain-mappings create \
  --service=hauslet-api \
  --domain=api.hauslet.com \
  --region=$REGION

# Follow instructions to configure DNS records
# Add CNAME record: api.hauslet.com -> ghs.googlehosted.com
```

### Step 9: Monitoring Setup

```bash
# View logs
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-api" \
  --limit=50 \
  --format=json

# Set up log-based metrics (via Console or Terraform)
# Configure alerting policies (via Console or Terraform)
```

---

## Cost Estimation (Monthly)

| Service | Configuration | Estimated Cost |
|---------|--------------|----------------|
| Cloud Run (API) | 2 vCPU, 2GB RAM, 1M requests/month | $50 |
| Cloud Run (Worker) | 2 vCPU, 2GB RAM, 500K requests/month | $25 |
| Cloud SQL | db-custom-2-8192 (2 vCPU, 8GB), HA | $250 |
| Valkey (managed) | 5GB, HA | $120 |
| Cloud Tasks | 1M tasks/month | $0.40 |
| Cloud Scheduler | 4 jobs | $0.40 |
| Cloud Build | 120 builds/month | $12 |
| Artifact Registry | 50GB storage | $5 |
| Networking (VPC, egress) | ~100GB egress | $12 |
| Cloud Logging | 50GB logs | $25 |
| Secret Manager | 11 secrets, 10K accesses | $0.06 |
| **Total** | | **~$500/month** |

**Cost Optimization Tips:**
- Use Cloud Run minimum instances = 0 for Worker (cold starts acceptable)
- Enable Cloud SQL automatic storage increase only
- Set log retention to 30 days
- Use committed use discounts for Cloud SQL (save 30%)
- Monitor and right-size resources after first month

---

## Monitoring and Alerting

### Cloud Monitoring Dashboards

Create dashboards for:

1. **API Service**
   - Request rate, latency (p50, p95, p99)
   - Error rate (4xx, 5xx)
   - CPU and memory utilization
   - Active instances

2. **Worker Service**
   - Task processing rate
   - Task success/failure rate
   - Task latency
   - Active instances

3. **Database**
   - Connection count
   - Query latency
   - CPU and memory usage
   - Disk I/O

4. **Redis**
   - Hit/miss ratio
   - Memory usage
   - Connection count

### Alert Policies (Recommended)

```yaml
# Examples - implement via Terraform or Console

- name: "API Error Rate High"
  condition: error_rate > 5% for 5 minutes
  notification_channels: [pagerduty, slack]

- name: "Worker Task Failure Rate High"
  condition: task_failure_rate > 10% for 10 minutes
  notification_channels: [slack]

- name: "Database Connection Pool Exhausted"
  condition: active_connections > 90 for 2 minutes
  notification_channels: [pagerduty]

- name: "Cloud Tasks Queue Backlog"
  condition: queue_depth > 1000 for 15 minutes
  notification_channels: [slack]

- name: "Cloud SQL High CPU"
  condition: cpu_utilization > 80% for 10 minutes
  notification_channels: [slack]
```

---

## Testing Strategy

### Pre-Deployment Testing

1. **Local Docker Testing**
   ```bash
   # Build images locally
  docker build -f deploy/Dockerfile.api -t hauslet-api:local .
  docker build -f deploy/Dockerfile.worker -t hauslet-worker:local .

   # Run with docker-compose (update docker-compose.yml)
   docker-compose up
   ```

2. **Health Check Verification**
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/health/ready
   ```

### Staging Environment Testing

1. **Smoke Tests**
   - All health endpoints return 200
   - Database connectivity works
   - Redis connectivity works
   - Cloud Tasks queues exist

2. **Integration Tests**
   - Email queue → Worker processing
   - Media thumbnail generation
   - Booking expiry cron → queue → handler
   - Payout processing end-to-end

3. **Load Testing**
   ```bash
   # Using Apache Bench
   ab -n 1000 -c 10 https://staging-api.hauslet.com/api/v1/properties

   # Using k6
   k6 run load-test.js
   ```

### Production Rollout Strategy

1. **Blue-Green Deployment**
   - Deploy new version with tag (e.g., `v2`)
   - Route 10% traffic to new version
   - Monitor error rates and latency for 1 hour
   - Gradually increase to 50%, then 100%

2. **Rollback Plan**
   ```bash
   # Instant rollback to previous revision
   gcloud run services update-traffic hauslet-api \
     --to-revisions=hauslet-api-00015-abc=100 \
     --region=$REGION
   ```

---

## Migration Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Database migration race condition** | **Critical** | **Use `RunMigrations` with advisory lock to serialize concurrent startups** |
| **Job timeout failures** | **High** | **Configure 300s Cloud Run timeout, per-queue Cloud Tasks timeouts, remove hard 15s limit** |
| **Queue fallback masking infra issues** | **Medium** | **Implement environment-aware fallback (fail-fast in prod)** |
| NATS → Cloud Tasks behavioral differences | High | Extensive integration testing, gradual rollout, keep NATS running in parallel during migration |
| Cold start latency on Cloud Run | Medium | Set min instances = 1 for API, use warm-up requests, optimize container size |
| Cloud SQL connection limits | Medium | Use connection pooling, monitor active connections, configure max connections |
| Cloud Tasks rate limits | Low | Configure per-queue rate limits, implement exponential backoff |
| Scheduler timing drift | Low | Use idempotency keys, handle duplicate jobs gracefully |
| Cost overruns | Medium | Set budget alerts at $400, $500, $600, use autoscaling limits |
| Secret Manager access latency | Low | Cache secrets at startup, refresh periodically |
| VPC networking complexity | Medium | Use Terraform, test VPC connector before production |

---

## Success Criteria

Deployment is considered successful when:

- ✅ All 9 Cloud Tasks queues are processing jobs correctly
- ✅ All 4 Cloud Scheduler jobs are triggering on schedule
- ✅ API responds to health checks with <200ms latency
- ✅ Worker processes tasks with >95% success rate
- ✅ Worker can handle long-running tasks (AI moderation up to 300s)
- ✅ Database migrations are serialized by advisory lock (no race conditions)
- ✅ Database query latency <50ms (p95)
- ✅ Redis cache hit rate >80%
- ✅ Queue failures trigger alerts in production (fail-fast enabled)
- ✅ Zero downtime during deployment
- ✅ Cost within $500-600/month budget
- ✅ CI/CD pipeline deploys in <10 minutes
- ✅ All production secrets secured in Secret Manager

---

## Next Steps After Deployment

1. **Performance Optimization**
   - Enable Cloud CDN for static assets
   - Implement database query caching
   - Add Redis read replicas if needed
   - Optimize container image sizes

2. **Security Hardening**
   - Enable Cloud Armor (DDoS protection)
   - Implement VPC Service Controls
   - Set up Cloud IAM conditions
   - Enable audit logging
   - Configure Security Command Center

3. **Cost Optimization**
   - Analyze Cloud Billing reports weekly
   - Right-size Cloud Run instances
   - Consider committed use discounts for Cloud SQL
   - Implement log sampling for high-volume logs
   - Use Cloud Storage lifecycle policies

4. **Observability Enhancement**
   - Implement Cloud Trace (distributed tracing)
   - Add custom metrics for business KPIs
   - Set up Error Reporting integration
   - Implement log-based metrics
   - Create SLO/SLI dashboards

5. **Disaster Recovery**
   - Set up Cloud SQL cross-region replicas
   - Implement automated backup verification
   - Document disaster recovery runbook
   - Test restore procedures quarterly
   - Set up cross-region failover

---

## Troubleshooting

### Common Issues

**Issue**: Cloud Run service won't start
```bash
# Check logs
gcloud logging read "resource.type=cloud_run_revision" --limit=50

# Check service configuration
gcloud run services describe hauslet-api --region=$REGION

# Verify secrets are accessible
gcloud secrets versions access latest --secret=DB_PASSWORD
```

**Issue**: Cloud SQL connection fails
```bash
# Test connection via Cloud SQL Proxy
cloud_sql_proxy -instances=$INSTANCE_CONNECTION_NAME=tcp:5432

# Check IAM permissions
gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:hauslet-api-sa@$PROJECT_ID.iam.gserviceaccount.com"
```

**Issue**: Cloud Tasks not being delivered
```bash
# Check queue configuration
gcloud tasks queues describe email-queue --location=$REGION

# List recent tasks
gcloud tasks list --queue=email-queue --location=$REGION

# Check worker service logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-worker"
```

**Issue**: High costs
```bash
# Analyze costs by service
gcloud billing accounts list
gcloud billing accounts describe BILLING_ACCOUNT_ID

# View detailed billing report in Console:
# https://console.cloud.google.com/billing/
```

---

## Support and Resources

### Documentation
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud SQL Documentation](https://cloud.google.com/sql/docs)
- [Cloud Tasks Documentation](https://cloud.google.com/tasks/docs)
- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)

### Terraform Modules
- [Google Cloud Run Module](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_service)
- [Google Cloud SQL Module](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance)

### Community
- [Google Cloud Community](https://www.googlecloudcommunity.com/)
- [Stack Overflow - google-cloud-platform](https://stackoverflow.com/questions/tagged/google-cloud-platform)

---

**Last Updated**: 2025-12-27
**Migration Timeline**: 4-5 weeks
**Estimated Cost**: ~$500/month
**Team Size**: 2-3 engineers (Backend, DevOps, QA)
