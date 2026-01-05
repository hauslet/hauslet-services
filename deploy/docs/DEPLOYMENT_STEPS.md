# Quick Deployment Guide for Hauslet on GCP

Your current setup:
- ✅ Project: `gen-lang-client-0265949535`
- ✅ Region: `europe-north1`
- ✅ Artifact Registry: Created
- ✅ Secrets: Added via console
- ✅ Cloud SQL: Created
- ✅ Valkey/Redis: Created

## Option 1: Automated Deployment (Recommended)

Run the interactive deployment script:

```bash
cd deploy
./deploy-to-gcp.sh
```

This will guide you through:
1. ✅ Enabling required APIs
2. ✅ Creating service accounts
3. ✅ Granting IAM permissions
4. ✅ Setting up VPC connector
5. ✅ First deployment
6. ✅ Setting up GitHub triggers

**Time**: ~15-20 minutes (including 8-12 min build)

---

## Option 2: Manual Step-by-Step

### Step 1: Enable APIs

```bash
gcloud config set project gen-lang-client-0265949535

gcloud services enable \
    cloudbuild.googleapis.com \
    run.googleapis.com \
    cloudscheduler.googleapis.com \
    cloudtasks.googleapis.com \
    secretmanager.googleapis.com \
    sqladmin.googleapis.com \
    compute.googleapis.com \
    vpcaccess.googleapis.com
```

### Step 2: Create Service Accounts

```bash
# API Service Account
gcloud iam service-accounts create hauslet-api-sa \
    --display-name="Hauslet API Service Account"

# Worker Service Account
gcloud iam service-accounts create hauslet-worker-sa \
    --display-name="Hauslet Worker Service Account"
```

### Step 3: Grant Service Account Permissions

```bash
PROJECT_ID="gen-lang-client-0265949535"
API_SA="hauslet-api-sa@${PROJECT_ID}.iam.gserviceaccount.com"
WORKER_SA="hauslet-worker-sa@${PROJECT_ID}.iam.gserviceaccount.com"

# API Permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$API_SA" \
    --role="roles/cloudsql.client"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$API_SA" \
    --role="roles/secretmanager.secretAccessor"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$API_SA" \
    --role="roles/cloudtasks.enqueuer"

# Worker Permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$WORKER_SA" \
    --role="roles/cloudsql.client"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$WORKER_SA" \
    --role="roles/secretmanager.secretAccessor"
```

### Step 4: Grant Cloud Build Permissions

```bash
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

# Grant Cloud Run Admin
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/run.admin"

# Grant Service Account User
gcloud iam service-accounts add-iam-policy-binding $API_SA \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/iam.serviceAccountUser"

gcloud iam service-accounts add-iam-policy-binding $WORKER_SA \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/iam.serviceAccountUser"

# Grant Artifact Registry Access
gcloud artifacts repositories add-iam-policy-binding hauslet \
    --location=europe-north1 \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/artifactregistry.writer"
```

### Step 5: Create VPC Connector

Cloud Run needs this to access Cloud SQL and Redis:

```bash
gcloud compute networks vpc-access connectors create hauslet-vpc-connector \
    --region=europe-north1 \
    --network=default \
    --range=10.8.0.0/28 \
    --min-throughput=200 \
    --max-throughput=300
```

**Note**: This takes ~2 minutes to create.

### Step 6: Update cloudbuild.yaml

Get your Cloud SQL connection name:

```bash
gcloud sql instances list
# Note the instance name

# Your connection name will be:
# gen-lang-client-0265949535:europe-north1:YOUR_INSTANCE_NAME
```

Edit `cloudbuild.yaml` and update these lines in both deploy-api and deploy-worker steps:

```yaml
--set-cloudsql-instances=gen-lang-client-0265949535:europe-north1:YOUR_SQL_INSTANCE
--vpc-connector=hauslet-vpc-connector
```

### Step 7: First Deployment

```bash
# From project root
gcloud builds submit \
    --config=cloudbuild.yaml \
    --region=europe-north1
```

This will:
1. Build API and Worker Docker images
2. Push to Artifact Registry
3. Deploy to Cloud Run
4. Run smoke tests

**Time**: ~8-12 minutes

### Step 8: Verify Deployment

```bash
# Get API URL
API_URL=$(gcloud run services describe hauslet-api \
    --region=europe-north1 \
    --format='value(status.url)')

echo "API URL: $API_URL"

# Test health endpoint
curl $API_URL/health
# Expected: "ok"

# List deployed services
gcloud run services list --region=europe-north1
```

### Step 9: Set Up GitHub Triggers (Optional)

For automatic deployments on push:

1. **Connect GitHub**:
   - Go to: https://console.cloud.google.com/cloud-build/triggers/connect?project=gen-lang-client-0265949535
   - Select "GitHub (Cloud Build GitHub App)"
   - Authenticate and select your repository

2. **Create Production Trigger**:
```bash
gcloud builds triggers create github \
    --name="hauslet-production" \
    --region=europe-north1 \
    --repo-owner=YOUR_GITHUB_USERNAME \
    --repo-name=hauslet-services \
    --branch-pattern="^main$" \
    --build-config=cloudbuild.yaml
```

3. **Create Staging Trigger** (optional):
```bash
gcloud builds triggers create github \
    --name="hauslet-staging" \
    --region=europe-north1 \
    --repo-owner=YOUR_GITHUB_USERNAME \
    --repo-name=hauslet-services \
    --branch-pattern="^staging$" \
    --build-config=cloudbuild-staging.yaml
```

Now every push to `main` will auto-deploy!

---

## Verification Checklist

After deployment, verify:

- [ ] APIs enabled
- [ ] Service accounts created
- [ ] IAM permissions granted
- [ ] VPC connector created
- [ ] First build successful
- [ ] API service deployed
- [ ] Worker service deployed
- [ ] Health check returns 200
- [ ] GitHub triggers created (optional)

## Troubleshooting

### Build Fails: Permission Denied

If you see permission errors during build:

```bash
# Re-grant Cloud Build permissions
PROJECT_NUMBER=$(gcloud projects describe gen-lang-client-0265949535 --format="value(projectNumber)")
CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

gcloud projects add-iam-policy-binding gen-lang-client-0265949535 \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/run.admin"
```

### Health Check Fails

Check service logs:

```bash
gcloud logging read \
    "resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-api" \
    --limit=50 \
    --format=json
```

Common causes:
- Missing secrets (check Secret Manager IAM)
- Wrong Cloud SQL connection name
- Database connection failed

### VPC Connector Issues

List connectors:

```bash
gcloud compute networks vpc-access connectors list \
    --region=europe-north1
```

If missing, create it per Step 5.

## Next Steps After Deployment

1. **Test API Endpoints**:
   ```bash
   curl $API_URL/api/v1/properties
   ```

2. **Deploy Cloud Scheduler** (for periodic jobs):
   ```bash
   cd deploy/terraform
   # Update terraform.tfvars with worker URL
   ./deploy-scheduler.sh
   ```

3. **Set Up Monitoring**:
   - Cloud Run metrics
   - Log-based alerts
   - Uptime checks

4. **Configure Custom Domain** (optional):
   ```bash
   gcloud run domain-mappings create \
       --service=hauslet-api \
       --domain=api.yourdomain.com \
       --region=europe-north1
   ```

## Useful Commands

```bash
# View recent builds
gcloud builds list --region=europe-north1 --limit=5

# View build logs
gcloud builds log <BUILD_ID> --region=europe-north1

# List Cloud Run services
gcloud run services list --region=europe-north1

# View service logs
gcloud logging read "resource.type=cloud_run_revision" --limit=20

# Rollback to previous revision
gcloud run services update-traffic hauslet-api \
    --to-revisions=hauslet-api-00042-abc=100 \
    --region=europe-north1
```

## Support

- [Cloud Build Docs](https://cloud.google.com/build/docs)
- [Cloud Run Docs](https://cloud.google.com/run/docs)
- [Troubleshooting Guide](./CICD_GUIDE.md#troubleshooting)
