# gcloud CLI Cheatsheet

Quick reference for Google Cloud CLI commands used in Hauslet Services.

## Authentication & Configuration

```bash
# Login
gcloud auth login

# Application default credentials (for Terraform)
gcloud auth application-default login

# List accounts
gcloud auth list

# Set active account
gcloud config set account EMAIL

# Set project
gcloud config set project PROJECT_ID

# Set default region
gcloud config set run/region europe-north1
gcloud config set compute/region europe-north1

# Show current config
gcloud config list

# Create named configuration
gcloud config configurations create staging
gcloud config configurations activate staging
```

## Projects

```bash
# List projects
gcloud projects list

# Get project info
gcloud projects describe PROJECT_ID

# Get project number
gcloud projects describe PROJECT_ID --format="value(projectNumber)"

# Enable APIs
gcloud services enable run.googleapis.com
gcloud services enable cloudbuild.googleapis.com
gcloud services enable sqladmin.googleapis.com

# List enabled services
gcloud services list --enabled
```

## Cloud Run

### Services

```bash
# List services
gcloud run services list
gcloud run services list --region=europe-north1

# Describe service
gcloud run services describe SERVICE_NAME --region=REGION

# Get service URL
gcloud run services describe SERVICE_NAME \
  --region=REGION \
  --format='value(status.url)'

# Deploy service
gcloud run deploy SERVICE_NAME \
  --image=IMAGE_URL \
  --region=REGION \
  --platform=managed \
  --allow-unauthenticated

# Update service (scale, memory, etc.)
gcloud run services update SERVICE_NAME \
  --region=REGION \
  --min-instances=1 \
  --max-instances=10 \
  --cpu=2 \
  --memory=2Gi \
  --timeout=60s \
  --concurrency=80

# Delete service
gcloud run services delete SERVICE_NAME --region=REGION

# Add environment variable
gcloud run services update SERVICE_NAME \
  --region=REGION \
  --set-env-vars="KEY=value,KEY2=value2"

# Bind secret
gcloud run services update SERVICE_NAME \
  --region=REGION \
  --set-secrets="ENV_VAR=SECRET_NAME:latest"

# Set Cloud SQL connection
gcloud run services update SERVICE_NAME \
  --region=REGION \
  --set-cloudsql-instances="PROJECT:REGION:INSTANCE"
```

### IAM for Cloud Run

```bash
# Allow unauthenticated access
gcloud run services add-iam-policy-binding SERVICE_NAME \
  --region=REGION \
  --member="allUsers" \
  --role="roles/run.invoker"

# Allow specific service account
gcloud run services add-iam-policy-binding SERVICE_NAME \
  --region=REGION \
  --member="serviceAccount:EMAIL" \
  --role="roles/run.invoker"

# Get IAM policy
gcloud run services get-iam-policy SERVICE_NAME --region=REGION
```

### Logs

```bash
# Read recent logs
gcloud run services logs read SERVICE_NAME \
  --region=REGION \
  --limit=50

# Follow logs (tail -f)
gcloud run services logs tail SERVICE_NAME \
  --region=REGION

# Filter by severity
gcloud run services logs read SERVICE_NAME \
  --region=REGION \
  --log-filter='severity>=ERROR'

# Filter by time
gcloud run services logs read SERVICE_NAME \
  --region=REGION \
  --log-filter='timestamp>="2024-01-01T00:00:00Z"'
```

## Cloud Build

```bash
# Submit build
gcloud builds submit \
  --config=cloudbuild.yaml \
  --region=REGION \
  --substitutions=_TAG=v1.0.0

# List builds
gcloud builds list --region=REGION --limit=10

# Get build status
gcloud builds describe BUILD_ID --region=REGION

# View build logs
gcloud builds log BUILD_ID --region=REGION

# Stream build logs
gcloud builds log BUILD_ID --region=REGION --stream

# Cancel build
gcloud builds cancel BUILD_ID --region=REGION

# List triggers
gcloud builds triggers list --region=REGION

# Create GitHub trigger
gcloud builds triggers create github \
  --name="deploy-staging" \
  --repo-name="hauslet-services" \
  --repo-owner="USERNAME" \
  --branch-pattern="^staging$" \
  --build-config="cloudbuild-staging.yaml" \
  --region=REGION
```

## Cloud Scheduler

```bash
# List jobs
gcloud scheduler jobs list
gcloud scheduler jobs list --location=REGION

# Describe job
gcloud scheduler jobs describe JOB_NAME --location=REGION

# Create HTTP job
gcloud scheduler jobs create http JOB_NAME \
  --location=REGION \
  --schedule="*/15 * * * *" \
  --uri="https://example.com/endpoint" \
  --http-method=POST \
  --headers="Content-Type=application/json" \
  --message-body='{"key":"value"}'

# Update job
gcloud scheduler jobs update http JOB_NAME \
  --location=REGION \
  --schedule="*/30 * * * *"

# Run job manually
gcloud scheduler jobs run JOB_NAME --location=REGION

# Pause job
gcloud scheduler jobs pause JOB_NAME --location=REGION

# Resume job
gcloud scheduler jobs resume JOB_NAME --location=REGION

# Delete job
gcloud scheduler jobs delete JOB_NAME --location=REGION

# List available locations
gcloud scheduler locations list
```

## Cloud SQL

```bash
# List instances
gcloud sql instances list

# Describe instance
gcloud sql instances describe INSTANCE_NAME

# Create instance
gcloud sql instances create INSTANCE_NAME \
  --database-version=POSTGRES_15 \
  --tier=db-custom-2-8192 \
  --region=REGION \
  --network=default

# Connect to instance
gcloud sql connect INSTANCE_NAME --user=USERNAME --database=DBNAME

# Get connection name
gcloud sql instances describe INSTANCE_NAME \
  --format="value(connectionName)"

# List databases
gcloud sql databases list --instance=INSTANCE_NAME

# Create database
gcloud sql databases create DATABASE_NAME --instance=INSTANCE_NAME

# List users
gcloud sql users list --instance=INSTANCE_NAME

# Create user
gcloud sql users create USERNAME \
  --instance=INSTANCE_NAME \
  --password=PASSWORD

# List backups
gcloud sql backups list --instance=INSTANCE_NAME

# Create backup
gcloud sql backups create --instance=INSTANCE_NAME

# Restore backup
gcloud sql backups restore BACKUP_ID \
  --backup-instance=INSTANCE_NAME

# Patch instance (update settings)
gcloud sql instances patch INSTANCE_NAME \
  --backup-start-time=03:00 \
  --enable-bin-log
```

## Secret Manager

```bash
# List secrets
gcloud secrets list

# Create secret from stdin
echo -n "secret-value" | gcloud secrets create SECRET_NAME --data-file=-

# Create secret from file
gcloud secrets create SECRET_NAME --data-file=/path/to/file

# Get secret value
gcloud secrets versions access latest --secret=SECRET_NAME

# Update secret
echo -n "new-value" | gcloud secrets versions add SECRET_NAME --data-file=-

# Delete secret
gcloud secrets delete SECRET_NAME

# Describe secret
gcloud secrets describe SECRET_NAME

# Grant access to secret
gcloud secrets add-iam-policy-binding SECRET_NAME \
  --member="serviceAccount:EMAIL" \
  --role="roles/secretmanager.secretAccessor"

# List versions
gcloud secrets versions list SECRET_NAME

# Destroy version (can't be undone!)
gcloud secrets versions destroy VERSION --secret=SECRET_NAME
```

## IAM & Service Accounts

```bash
# List service accounts
gcloud iam service-accounts list

# Create service account
gcloud iam service-accounts create SA_NAME \
  --display-name="Display Name" \
  --description="Description"

# Delete service account
gcloud iam service-accounts delete EMAIL

# Get service account email
gcloud iam service-accounts list \
  --filter="displayName:NAME" \
  --format="value(email)"

# Grant project role to service account
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:EMAIL" \
  --role="roles/ROLE"

# Remove role
gcloud projects remove-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:EMAIL" \
  --role="roles/ROLE"

# List project IAM policy
gcloud projects get-iam-policy PROJECT_ID

# Create service account key
gcloud iam service-accounts keys create key.json \
  --iam-account=EMAIL

# List keys
gcloud iam service-accounts keys list --iam-account=EMAIL
```

## Artifact Registry

```bash
# List repositories
gcloud artifacts repositories list

# Create repository
gcloud artifacts repositories create REPO_NAME \
  --repository-format=docker \
  --location=REGION \
  --description="Description"

# Delete repository
gcloud artifacts repositories delete REPO_NAME --location=REGION

# List images
gcloud artifacts docker images list REGION-docker.pkg.dev/PROJECT/REPO

# Delete image
gcloud artifacts docker images delete IMAGE_URL

# Grant access
gcloud artifacts repositories add-iam-policy-binding REPO_NAME \
  --location=REGION \
  --member="serviceAccount:EMAIL" \
  --role="roles/artifactregistry.writer"

# Configure docker
gcloud auth configure-docker REGION-docker.pkg.dev
```

## Storage (GCS)

```bash
# List buckets
gsutil ls

# Create bucket
gsutil mb -l REGION gs://BUCKET_NAME

# Delete bucket
gsutil rm -r gs://BUCKET_NAME

# List objects in bucket
gsutil ls gs://BUCKET_NAME

# Copy file to bucket
gsutil cp file.txt gs://BUCKET_NAME/

# Copy from bucket
gsutil cp gs://BUCKET_NAME/file.txt .

# Make bucket public
gsutil iam ch allUsers:objectViewer gs://BUCKET_NAME

# Set lifecycle policy
gsutil lifecycle set lifecycle.json gs://BUCKET_NAME

# Enable versioning
gsutil versioning set on gs://BUCKET_NAME
```

## Logging

```bash
# Read logs
gcloud logging read "FILTER" --limit=50

# Read Cloud Run logs
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=SERVICE" \
  --limit=50

# Read error logs
gcloud logging read "severity>=ERROR" --limit=50

# Follow logs
gcloud logging tail "FILTER"

# List log entries
gcloud logging logs list

# Create sink
gcloud logging sinks create SINK_NAME \
  DESTINATION \
  --log-filter='FILTER'
```

## Useful Filters & Formats

### Common Filters

```bash
# Severity
severity>=ERROR
severity=WARNING

# Time
timestamp>="2024-01-01T00:00:00Z"
timestamp<"2024-01-02T00:00:00Z"

# Resource type
resource.type="cloud_run_revision"
resource.type="cloud_scheduler_job"

# Service name
resource.labels.service_name="hauslet-api-staging"

# HTTP status
httpRequest.status=500

# Combine filters
resource.type="cloud_run_revision" AND severity>=ERROR AND timestamp>="2024-01-01T00:00:00Z"
```

### Common Formats

```bash
# JSON output
--format=json

# Specific field
--format="value(status.url)"
--format="value(email)"

# Table with specific columns
--format="table(name,schedule,state)"

# CSV
--format=csv

# YAML
--format=yaml
```

## Productivity Tips

### Aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Quick deployment
alias deploy-staging='cd ~/hauslet-services/deploy && ./deploy-staging.sh'

# Log viewers
alias logs-api='gcloud run services logs read hauslet-api-staging --region=europe-north1 --limit=50'
alias logs-worker='gcloud run services logs read hauslet-worker-staging --region=europe-north1 --limit=50'

# Scheduler
alias sched-list='gcloud scheduler jobs list --location=europe-west1'
alias sched-run='gcloud scheduler jobs run'

# Quick status
alias status='gcloud run services list --region=europe-north1'
```

### Environment Variables

```bash
# Set common values
export PROJECT_ID="gen-lang-client-0265949535"
export REGION="europe-north1"

# Use in commands
gcloud run services list --region=$REGION
```

### Configuration Sets

```bash
# Create configs for each environment
gcloud config configurations create staging
gcloud config set project gen-lang-client-0265949535
gcloud config set run/region europe-north1

gcloud config configurations create production
gcloud config set project hauslet-prod
gcloud config set run/region us-central1

# Switch between configs
gcloud config configurations activate staging
gcloud config configurations activate production
```

### Output Processing with jq

```bash
# Install jq: brew install jq (macOS)

# Get service URLs
gcloud run services list --region=europe-north1 --format=json | \
  jq -r '.[] | "\(.metadata.name): \(.status.url)"'

# Get all secrets
gcloud secrets list --format=json | jq -r '.[].name'

# Get error count from logs
gcloud logging read "severity=ERROR" --format=json --limit=1000 | \
  jq 'length'
```

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│                    Most Used Commands                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Configuration:                                              │
│  gcloud config set project PROJECT_ID                        │
│  gcloud config set run/region REGION                         │
│                                                               │
│  Cloud Run:                                                  │
│  gcloud run services list --region=REGION                    │
│  gcloud run services logs read SERVICE --region=REGION       │
│  gcloud run deploy SERVICE --image=IMAGE --region=REGION     │
│                                                               │
│  Cloud Build:                                                │
│  gcloud builds submit --config=cloudbuild.yaml               │
│  gcloud builds list --region=REGION                          │
│                                                               │
│  Scheduler:                                                  │
│  gcloud scheduler jobs list --location=REGION                │
│  gcloud scheduler jobs run JOB --location=REGION             │
│                                                               │
│  Secrets:                                                    │
│  echo -n "value" | gcloud secrets create NAME --data-file=-  │
│  gcloud secrets versions access latest --secret=NAME         │
│                                                               │
│  IAM:                                                        │
│  gcloud projects add-iam-policy-binding PROJECT_ID \         │
│    --member=serviceAccount:EMAIL --role=roles/ROLE           │
│                                                               │
│  Logs:                                                       │
│  gcloud logging read "severity>=ERROR" --limit=50            │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Resources

- [gcloud CLI Documentation](https://cloud.google.com/sdk/gcloud/reference)
- [Cloud Run Reference](https://cloud.google.com/sdk/gcloud/reference/run)
- [Cloud Build Reference](https://cloud.google.com/sdk/gcloud/reference/builds)
- [IAM Reference](https://cloud.google.com/sdk/gcloud/reference/iam)
- [Logging Filters](https://cloud.google.com/logging/docs/view/logging-query-language)
