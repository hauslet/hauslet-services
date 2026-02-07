# CI/CD Pipeline Guide for Hauslet Services

Complete guide for setting up and using the automated deployment pipeline with Cloud Build.

## Overview

### What's Automated

- ✅ **Building**: Automatically builds Docker images for API and Worker
- ✅ **Testing**: Runs smoke tests after deployment
- ✅ **Deploying**: Deploys to Cloud Run (staging and production)
- ✅ **Versioning**: Tags images with commit SHA and environment
- ✅ **Traffic Management**: Gradual rollout with traffic splitting

### Architecture

```
GitHub Push (main/staging)
        ↓
Cloud Build Trigger
        ↓
Build Docker Images → Push to Artifact Registry
        ↓
Deploy to Cloud Run
        ↓
Run Smoke Tests
        ↓
Update Traffic
```

## Prerequisites

Before setting up CI/CD:

1. **GCP Project** with billing enabled
2. **GitHub Repository** with code access
3. **Local Tools**:
   - `gcloud` CLI installed and authenticated
   - `terraform` v1.5+
   - Bash shell

## Quick Start

### Step 1: Initial Setup

```bash
# Set your project
export PROJECT_ID="hauslet-prod"
export GITHUB_OWNER="your-github-org"
export REGION="us-central1"

# Run automated setup
cd deploy
./setup-cicd.sh
```

This script will:
- ✅ Enable required GCP APIs
- ✅ Create Artifact Registry repository
- ✅ Create service accounts with proper IAM permissions
- ✅ Set up storage bucket for build artifacts
- ✅ Configure Cloud Build triggers

### Step 2: Configure Secrets

```bash
# Configure all required secrets
PROJECT_ID=hauslet-prod ./configure-secrets.sh
```

You'll be prompted for:
- Database passwords
- API keys (Flutterwave, Paystack, Gemini, etc.)
- OAuth secrets
- Encryption keys

### Step 3: Connect GitHub

1. Go to [Cloud Build Triggers](https://console.cloud.google.com/cloud-build/triggers)
2. Click "Connect Repository"
3. Select "GitHub (Cloud Build GitHub App)"
4. Authenticate and select your repository
5. Note the connection name

### Step 4: Deploy Infrastructure

```bash
cd deploy/terraform

# Review and apply full infrastructure
terraform init
terraform plan
terraform apply
```

This creates:
- Cloud SQL PostgreSQL
- Memorystore Redis (or Valkey)
- VPC and connectors
- Cloud Tasks queues
- Cloud Scheduler jobs

### Step 5: First Deployment

Trigger your first deployment:

```bash
# Option 1: Push to main branch (automatic)
git push origin main

# Option 2: Manual trigger
gcloud builds submit \
    --config=cloudbuild.yaml \
    --region=us-central1
```

## Branching Strategy

### Production Branch: `main`

- **Trigger**: `cloudbuild.yaml`
- **Environment**: Production
- **Auto-deploy**: Yes (on push to main)
- **Services**: `hauslet-api`, `hauslet-worker`
- **URL**: `https://hauslet-api-xxx.a.run.app`

### Staging Branch: `staging`

- **Trigger**: `cloudbuild-staging.yaml`
- **Environment**: Staging
- **Auto-deploy**: Yes (on push to staging)
- **Services**: `hauslet-api-staging`, `hauslet-worker-staging`
- **URL**: `https://hauslet-api-staging-xxx.a.run.app`

### Workflow

```bash
# Feature development
git checkout -b feature/new-feature
git commit -m "Add new feature"

# Deploy to staging first
git checkout staging
git merge feature/new-feature
git push origin staging  # Auto-deploys to staging

# After testing, promote to production
git checkout main
git merge staging
git push origin main  # Auto-deploys to production
```

## Build Process

### cloudbuild.yaml Steps

1. **Build API Image** (parallel)
   - Uses `deploy/Dockerfile.api`
   - Tags: `{commit-sha}`, `latest`

2. **Build Worker Image** (parallel)
   - Uses `deploy/Dockerfile.worker`
   - Tags: `{commit-sha}`, `latest`

3. **Push to Artifact Registry** (after builds)
   - Location: `us-central1-docker.pkg.dev/{project}/hauslet/`

4. **Deploy API to Cloud Run**
   - Timeout: 60s
   - Min instances: 1 (prod), 0 (staging)
   - Max instances: 10 (prod), 3 (staging)

5. **Deploy Worker to Cloud Run**
   - Timeout: 300s (5 minutes for long tasks)
   - Min instances: 0
   - Max instances: 5 (prod), 2 (staging)

6. **Run Smoke Tests**
   - Test `/health` endpoint
   - Test `/health/ready` endpoint
   - Verify service status

7. **Update Traffic**
   - Route 100% traffic to new revision
   - Old revisions kept for rollback

### Build Time

Typical build times:
- **API Build**: ~3-5 minutes
- **Worker Build**: ~3-5 minutes
- **Total Pipeline**: ~8-12 minutes

## Monitoring Deployments

### View Build Status

```bash
# List recent builds
gcloud builds list --region=us-central1 --limit=10

# View specific build
BUILD_ID="abc-123-def"
gcloud builds describe $BUILD_ID --region=us-central1

# Stream logs for running build
gcloud builds log --stream $BUILD_ID --region=us-central1
```

### View Deployed Services

```bash
# List Cloud Run services
gcloud run services list --region=us-central1

# Describe specific service
gcloud run services describe hauslet-api \
    --region=us-central1

# Get service URL
gcloud run services describe hauslet-api \
    --region=us-central1 \
    --format='value(status.url)'
```

### View Build History

Via Cloud Console:
1. Go to [Cloud Build History](https://console.cloud.google.com/cloud-build/builds)
2. Filter by trigger name
3. Click build for detailed logs

## Rollback Procedures

### Quick Rollback (Traffic Split)

```bash
# List revisions
gcloud run revisions list \
    --service=hauslet-api \
    --region=us-central1

# Route traffic back to previous revision
PREVIOUS_REVISION="hauslet-api-00042-abc"
gcloud run services update-traffic hauslet-api \
    --region=us-central1 \
    --to-revisions=$PREVIOUS_REVISION=100
```

### Rollback to Specific Commit

```bash
# Find commit hash of working version
git log --oneline

# Trigger build for that commit
git checkout abc123
gcloud builds submit \
    --config=cloudbuild.yaml \
    --region=us-central1
```

### Emergency Rollback

```bash
# Deploy previous image directly
PREVIOUS_IMAGE="us-central1-docker.pkg.dev/hauslet-prod/hauslet/hauslet-api:abc123"

gcloud run deploy hauslet-api \
    --image=$PREVIOUS_IMAGE \
    --region=us-central1
```

## Troubleshooting

### Build Fails: Permission Denied

**Symptom**: `Permission denied` errors during Cloud Build

**Solution**:
```bash
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

# Grant necessary permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/run.admin"
```

### Build Fails: Image Push Error

**Symptom**: `failed to push image to Artifact Registry`

**Solution**:
```bash
# Grant Artifact Registry writer permission
gcloud artifacts repositories add-iam-policy-binding hauslet \
    --location=us-central1 \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/artifactregistry.writer"
```

### Deployment Fails: Health Check

**Symptom**: Service deploys but fails health checks

**Debug**:
```bash
# View deployment logs
gcloud logging read \
    "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-api" \
    --limit=50

# Check service configuration
gcloud run services describe hauslet-api \
    --region=us-central1
```

**Common causes**:
- Database connection issues (check Cloud SQL connection)
- Missing secrets (verify Secret Manager access)
- Wrong environment variables

### Build Timeout

**Symptom**: Build exceeds 20 minute timeout

**Solution**: Optimize Dockerfile
```dockerfile
# Use multi-stage builds
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download  # Cached layer
COPY . .
RUN CGO_ENABLED=0 go build -o api cmd/api/main.go

FROM alpine:latest
COPY --from=builder /app/api /api
ENTRYPOINT ["/api"]
```

### Secrets Not Accessible

**Symptom**: Service can't read secrets from Secret Manager

**Solution**:
```bash
# Grant service account access
API_SA="hauslet-api-sa@$PROJECT_ID.iam.gserviceaccount.com"

gcloud secrets add-iam-policy-binding DB_PASSWORD \
    --member="serviceAccount:$API_SA" \
    --role="roles/secretmanager.secretAccessor"
```

## Cost Optimization

### Build Costs

**Cloud Build Pricing**: First 120 build-minutes/day free, then $0.003/build-minute

**Optimize**:
- Use Docker layer caching
- Parallel builds where possible
- Use `E2_HIGHCPU_8` for faster builds (costs more per minute but total time is less)

**Estimated monthly costs**:
- ~30 deployments/month × 10 minutes = 300 build-minutes
- First 120 minutes free × 30 days = 3,600 free minutes
- **Cost**: $0/month (within free tier)

### Artifact Registry Costs

**Pricing**: $0.10/GB/month for storage

**Optimize**:
- Use cleanup policies (keep last 20 tagged images)
- Delete untagged images after 30 days

**Estimated monthly costs**:
- ~20 images × 500MB = 10GB
- **Cost**: ~$1/month

## Best Practices

### 1. Semantic Versioning

Tag releases for production:
```bash
git tag -a v1.2.3 -m "Release version 1.2.3"
git push origin v1.2.3
```

### 2. Deployment Notes

Add deployment notes in commit messages:
```bash
git commit -m "feat: Add user authentication

Deployment notes:
- Requires DB_AUTH_SECRET secret
- Run migration: 003_add_auth_tables.sql
- Update API_VERSION to 2.0
"
```

### 3. Test Before Production

Always deploy to staging first:
```bash
git push origin staging  # Test here first
# Verify staging works
git push origin main     # Then deploy to prod
```

### 4. Monitor After Deployment

Check logs for 10 minutes after deploy:
```bash
gcloud logging tail \
    "resource.type=cloud_run_revision" \
    --format=json
```

### 5. Database Migrations

The API runs `database.RunMigrations` on startup with advisory lock (safe for concurrent instances). No separate migration step needed.

## Advanced Configuration

### Custom Build Substitutions

Override default values:
```bash
gcloud builds submit \
    --config=cloudbuild.yaml \
    --substitutions=_API_MIN_INSTANCES=2,_WORKER_MAX_INSTANCES=10
```

### Conditional Deployments

Skip deployment for specific commits:
```bash
git commit -m "docs: Update README [skip ci]"
```

### Blue-Green Deployments

Deploy new version with no traffic:
```bash
# Deploy new revision with tag (no traffic)
gcloud run deploy hauslet-api \
    --image=...
    --no-traffic \
    --tag=blue

# Test blue revision
curl https://blue---hauslet-api-xxx.a.run.app/health

# Switch traffic
gcloud run services update-traffic hauslet-api \
    --to-tags=blue=100
```

## Next Steps

After CI/CD is working:

1. ✅ Set up monitoring dashboards
2. ✅ Configure alerting for failed deployments
3. ✅ Implement automated integration tests
4. ✅ Set up deployment notifications (Slack/email)
5. ✅ Document rollback procedures for team

## Support

- [Cloud Build Documentation](https://cloud.google.com/build/docs)
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Artifact Registry Documentation](https://cloud.google.com/artifact-registry/docs)
