# Deployment

This directory contains all deployment-related configurations, scripts, and documentation for deploying Hauslet Services to Google Cloud Platform (GCP).

## 📋 Quick Start

1. **Read the deployment plan**: [GCP_DEPLOYMENT_PLAN.md](./GCP_DEPLOYMENT_PLAN.md)
2. **Set up infrastructure**: Navigate to `terraform/` and follow instructions
3. **Configure CI/CD**: Use `cloudbuild.yaml` in project root
4. **Deploy**: Push to `main` branch (auto-deploys via Cloud Build)

## 📂 Directory Structure

```
/deploy
├── GCP_DEPLOYMENT_PLAN.md          # Comprehensive migration plan and deployment guide
├── terraform/                       # Infrastructure as Code (GCP resources)
│   ├── main.tf                     # Main Terraform configuration
│   ├── variables.tf                # Input variables
│   ├── outputs.tf                  # Output values
│   ├── vpc.tf                      # VPC and networking
│   ├── cloudsql.tf                 # Cloud SQL PostgreSQL
│   ├── redis.tf                    # Memorystore Redis
│   ├── cloudrun.tf                 # Cloud Run services
│   ├── cloudtasks.tf               # Cloud Tasks queues
│   ├── cloudscheduler.tf           # Cloud Scheduler jobs
│   └── iam.tf                      # Service accounts and IAM
├── scripts/                         # Deployment automation scripts
│   ├── setup-gcp.sh                # Initial GCP project setup
│   ├── create-secrets.sh           # Secret Manager setup
│   ├── migrate-db.sh               # Database migration helper
│   └── deploy-manual.sh            # Manual deployment (bypasses CI/CD)
└── kubernetes/                      # Future: If migrating to GKE
    └── (reserved for future use)
```

## 🏗️ Infrastructure

### Current Deployment Target: Google Cloud Platform

**Architecture**: Serverless (Cloud Run)

**Services**:
- **Cloud Run** - API and Worker services (auto-scaling, pay-per-request)
- **Cloud SQL for PostgreSQL** - Managed database (HA, automated backups)
- **Memorystore for Redis** - Managed cache (sessions + application cache)
- **Cloud Tasks** - Message queue (replaces NATS in production)
- **Cloud Scheduler** - Cron jobs (4 scheduled tasks)
- **Cloud Build** - CI/CD pipeline (automated deployments)
- **Artifact Registry** - Docker image storage
- **Secret Manager** - Credential management

**Estimated Monthly Cost**: ~$500 (see GCP_DEPLOYMENT_PLAN.md for breakdown)

## 🚀 Deployment Process

### Automatic Deployment (Recommended)

**Main Branch → Production**
```bash
git checkout main
git merge develop
git push origin main
# Cloud Build automatically deploys to production
```

**Staging Branch → Staging Environment**
```bash
git checkout staging
git merge develop
git push origin staging
# Cloud Build automatically deploys to staging
```

### Manual Deployment

```bash
# Build and deploy manually
cd deploy
./scripts/deploy-manual.sh production

# Or using gcloud directly
gcloud builds submit --config=../cloudbuild.yaml --region=us-central1
```

## 🔧 Initial Setup

### Prerequisites
- GCP Project with billing enabled
- gcloud CLI installed and authenticated
- Terraform v1.5+ installed
- GitHub repository connected to Cloud Build

### One-Time Setup

```bash
# 1. Set environment variables
export PROJECT_ID="hauslet-prod"
export REGION="us-central1"

# 2. Run GCP setup script
cd deploy/scripts
./setup-gcp.sh

# 3. Deploy infrastructure with Terraform
cd ../terraform
terraform init
terraform plan -var="project_id=$PROJECT_ID" -var="region=$REGION"
terraform apply

# 4. Create secrets
cd ../scripts
./create-secrets.sh

# 5. Run database migrations
./migrate-db.sh

# 6. Deploy application
cd ../..
gcloud builds submit --config=cloudbuild.yaml
```

## 📊 Monitoring and Logs

### View Logs
```bash
# API logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-api" --limit=50

# Worker logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-worker" --limit=50

# Cloud Tasks logs
gcloud logging read "resource.type=cloud_tasks_queue" --limit=50
```

### Monitoring Dashboards
- **Cloud Console**: https://console.cloud.google.com/monitoring
- **API Service**: Cloud Run → hauslet-api → Metrics
- **Worker Service**: Cloud Run → hauslet-worker → Metrics
- **Database**: Cloud SQL → hauslet-postgres → Metrics

## 🔐 Secrets Management

All secrets are stored in **Google Cloud Secret Manager**.

### List Secrets
```bash
gcloud secrets list
```

### Update a Secret
```bash
echo -n "new-secret-value" | gcloud secrets versions add SECRET_NAME --data-file=-
```

### Grant Access to Service Account
```bash
gcloud secrets add-iam-policy-binding SECRET_NAME \
  --member="serviceAccount:SERVICE_ACCOUNT@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

## 🧪 Testing

### Staging Environment
```bash
# Deploy to staging
git push origin staging

# Test staging endpoints
export STAGING_URL=$(gcloud run services describe hauslet-api-staging --region=us-central1 --format='value(status.url)')
curl $STAGING_URL/health
```

### Load Testing
```bash
# Using Apache Bench
ab -n 1000 -c 10 https://api.hauslet.com/api/v1/properties

# Using k6 (if configured)
k6 run load-test.js
```

## 🆘 Troubleshooting

### Service Won't Start
```bash
# Check logs
gcloud logging read "resource.type=cloud_run_revision" --limit=50

# Describe service
gcloud run services describe hauslet-api --region=us-central1
```

### Database Connection Issues
```bash
# Test via Cloud SQL Proxy
cloud_sql_proxy -instances=PROJECT:REGION:INSTANCE=tcp:5432

# Check IAM permissions
gcloud sql instances describe hauslet-postgres
```

### High Costs
```bash
# View billing dashboard
gcloud billing accounts list
# Then visit: https://console.cloud.google.com/billing/
```

## 📚 Documentation

- **[GCP_DEPLOYMENT_PLAN.md](./GCP_DEPLOYMENT_PLAN.md)** - Complete migration and deployment guide
- **[Terraform Docs](./terraform/README.md)** - Infrastructure as Code documentation
- **[Cloud Run Docs](https://cloud.google.com/run/docs)** - Google Cloud Run documentation
- **[Cloud SQL Docs](https://cloud.google.com/sql/docs)** - Google Cloud SQL documentation

## 🔄 Migration from NATS to Cloud Tasks

The application is migrating from NATS JetStream (local/dev) to Google Cloud Tasks (production). See GCP_DEPLOYMENT_PLAN.md for:
- Queue mappings (11 queues)
- Code changes required
- Migration strategy
- Rollback plan

## 📞 Support

- **Internal**: Slack #hauslet-devops
- **GCP Support**: https://cloud.google.com/support
- **Emergency Runbook**: See GCP_DEPLOYMENT_PLAN.md → Troubleshooting

---

**Last Updated**: 2025-12-27
**Cloud Platform**: Google Cloud Platform (GCP)
**Region**: us-central1
**Deployment Method**: Cloud Build + Terraform
