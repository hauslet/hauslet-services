#!/bin/bash
set -e

# Cloud Scheduler Deployment Script for Hauslet
# This script deploys the 4 Cloud Scheduler jobs that replace manual tickers

REGION="${REGION:-us-central1}"
PROJECT_ID="${PROJECT_ID:-}"

echo "🚀 Hauslet Cloud Scheduler Deployment"
echo "====================================="
echo ""

# Check prerequisites
if [ -z "$PROJECT_ID" ]; then
    echo "❌ ERROR: PROJECT_ID environment variable not set"
    echo "Usage: PROJECT_ID=hauslet-prod ./deploy-scheduler.sh"
    exit 1
fi

if ! command -v terraform &> /dev/null; then
    echo "❌ ERROR: Terraform not installed"
    echo "Install: https://developer.hashicorp.com/terraform/downloads"
    exit 1
fi

if ! command -v gcloud &> /dev/null; then
    echo "❌ ERROR: gcloud CLI not installed"
    echo "Install: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Set gcloud project
gcloud config set project "$PROJECT_ID"

echo "📋 Configuration:"
echo "   Project ID: $PROJECT_ID"
echo "   Region: $REGION"
echo ""

# Get Worker service URL
echo "🔍 Getting Worker service URL..."
WORKER_URL=$(gcloud run services describe hauslet-worker \
    --region="$REGION" \
    --format='value(status.url)' 2>/dev/null || echo "")

if [ -z "$WORKER_URL" ]; then
    echo "❌ ERROR: Worker service not found"
    echo "Please deploy the Worker service first:"
    echo "   gcloud run deploy hauslet-worker --region=$REGION ..."
    exit 1
fi

echo "   ✅ Worker URL: $WORKER_URL"

# Get service account
WORKER_SA=$(gcloud run services describe hauslet-worker \
    --region="$REGION" \
    --format='value(spec.template.spec.serviceAccountName)' 2>/dev/null || echo "")

if [ -z "$WORKER_SA" ]; then
    # Try default compute service account
    WORKER_SA="hauslet-worker-sa@${PROJECT_ID}.iam.gserviceaccount.com"
    echo "   ⚠️  Using default service account: $WORKER_SA"
else
    echo "   ✅ Service Account: $WORKER_SA"
fi

echo ""

# Check if terraform.tfvars exists
if [ ! -f "terraform.tfvars" ]; then
    echo "📝 Creating terraform.tfvars..."
    cat > terraform.tfvars <<EOF
project_id = "$PROJECT_ID"
region     = "$REGION"

worker_url             = "$WORKER_URL"
worker_service_name    = "hauslet-worker"
worker_service_account = "$WORKER_SA"

api_service_name    = "hauslet-api"
api_service_account = "hauslet-api-sa@${PROJECT_ID}.iam.gserviceaccount.com"

environment = "production"
EOF
    echo "   ✅ Created terraform.tfvars"
else
    echo "   ℹ️  Using existing terraform.tfvars"
fi

echo ""

# Initialize Terraform
echo "🔧 Initializing Terraform..."
terraform init

echo ""

# Plan
echo "📊 Planning deployment..."
terraform plan

echo ""
read -p "⚠️  Do you want to apply these changes? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Deployment cancelled"
    exit 0
fi

echo ""

# Apply
echo "🚀 Deploying Cloud Scheduler jobs..."
terraform apply -auto-approve

echo ""
echo "✅ Deployment complete!"
echo ""

# List created jobs
echo "📋 Created Cloud Scheduler Jobs:"
gcloud scheduler jobs list --location="$REGION" --filter="name:*-scheduler"

echo ""
echo "🧪 Testing Jobs..."
echo ""

# Test each job
JOBS=("media-cleanup-scheduler" "booking-expiry-scheduler" "booking-completion-scheduler" "payout-process-scheduler" "disbursement-retry-scheduler")

for JOB in "${JOBS[@]}"; do
    echo "   Testing $JOB..."
    if gcloud scheduler jobs run "$JOB" --location="$REGION" 2>/dev/null; then
        echo "   ✅ $JOB triggered successfully"
    else
        echo "   ⚠️  Failed to trigger $JOB (may need a few seconds to be ready)"
    fi
done

echo ""
echo "📊 View logs with:"
echo "   gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-worker' --limit=50"
echo ""
echo "✨ Phase 4: Cloud Scheduler Migration Complete!"
