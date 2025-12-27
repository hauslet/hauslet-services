#!/bin/bash
set -e

# CI/CD Setup Script for Hauslet Services
# Sets up Cloud Build, Artifact Registry, and triggers for automated deployments

PROJECT_ID="${PROJECT_ID:-}"
REGION="${REGION:-us-central1}"
GITHUB_OWNER="${GITHUB_OWNER:-}"
GITHUB_REPO="${GITHUB_REPO:-hauslet-services}"

echo "🚀 Hauslet CI/CD Setup"
echo "====================="
echo ""

# Validation
if [ -z "$PROJECT_ID" ]; then
    echo "❌ ERROR: PROJECT_ID not set"
    echo "Usage: PROJECT_ID=hauslet-prod GITHUB_OWNER=your-org ./setup-cicd.sh"
    exit 1
fi

if [ -z "$GITHUB_OWNER" ]; then
    echo "❌ ERROR: GITHUB_OWNER not set"
    echo "Usage: PROJECT_ID=hauslet-prod GITHUB_OWNER=your-org ./setup-cicd.sh"
    exit 1
fi

# Check prerequisites
if ! command -v gcloud &> /dev/null; then
    echo "❌ ERROR: gcloud not installed"
    exit 1
fi

if ! command -v terraform &> /dev/null; then
    echo "❌ ERROR: terraform not installed"
    exit 1
fi

gcloud config set project "$PROJECT_ID"

echo "📋 Configuration:"
echo "   Project: $PROJECT_ID"
echo "   Region: $REGION"
echo "   GitHub: $GITHUB_OWNER/$GITHUB_REPO"
echo ""

# Step 1: Enable required APIs
echo "🔧 Step 1: Enabling required APIs..."
gcloud services enable \
    cloudbuild.googleapis.com \
    artifactregistry.googleapis.com \
    run.googleapis.com \
    cloudscheduler.googleapis.com \
    cloudtasks.googleapis.com \
    secretmanager.googleapis.com \
    sqladmin.googleapis.com \
    compute.googleapis.com \
    vpcaccess.googleapis.com

echo "✅ APIs enabled"
echo ""

# Step 2: Deploy Terraform infrastructure
echo "🔧 Step 2: Deploying CI/CD infrastructure with Terraform..."
cd deploy/terraform

if [ ! -f "terraform.tfvars" ]; then
    echo "⚠️  terraform.tfvars not found. Creating from example..."
    cp terraform.tfvars.example terraform.tfvars
    echo "❌ Please edit terraform.tfvars and run this script again"
    exit 1
fi

terraform init
terraform plan -out=cicd.tfplan

read -p "Apply Terraform plan? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Cancelled"
    exit 0
fi

terraform apply cicd.tfplan

echo "✅ Terraform infrastructure deployed"
echo ""

# Get outputs
ARTIFACT_REGISTRY_URL=$(terraform output -raw artifact_registry_url 2>/dev/null || echo "")
API_SA=$(terraform output -raw api_service_account_email 2>/dev/null || echo "")
WORKER_SA=$(terraform output -raw worker_service_account_email 2>/dev/null || echo "")

echo "📦 Artifact Registry: $ARTIFACT_REGISTRY_URL"
echo "👤 API Service Account: $API_SA"
echo "👤 Worker Service Account: $WORKER_SA"
echo ""

cd ../..

# Step 3: Connect GitHub repository
echo "🔧 Step 3: Connecting GitHub repository..."
echo ""
echo "⚠️  You need to manually connect your GitHub repository:"
echo "   1. Go to: https://console.cloud.google.com/cloud-build/triggers/connect?project=$PROJECT_ID"
echo "   2. Select 'GitHub (Cloud Build GitHub App)'"
echo "   3. Authenticate and select repository: $GITHUB_OWNER/$GITHUB_REPO"
echo "   4. Note the connection name (e.g., 'hauslet-github')"
echo ""
read -p "Enter connection name (or press Enter to skip): " CONNECTION_NAME

if [ -z "$CONNECTION_NAME" ]; then
    echo "⚠️  Skipping trigger creation. You'll need to create triggers manually."
    echo ""
else
    # Step 4: Create Cloud Build triggers
    echo "🔧 Step 4: Creating Cloud Build triggers..."

    # Production trigger (main branch)
    echo "Creating production trigger..."
    gcloud builds triggers create github \
        --name="hauslet-production" \
        --region="$REGION" \
        --repo-name="$GITHUB_REPO" \
        --repo-owner="$GITHUB_OWNER" \
        --branch-pattern="^main$" \
        --build-config="cloudbuild.yaml" \
        --description="Deploy to production on main branch push" \
        --substitutions="_REGION=$REGION" || echo "⚠️  Trigger already exists or failed"

    # Staging trigger (staging branch)
    echo "Creating staging trigger..."
    gcloud builds triggers create github \
        --name="hauslet-staging" \
        --region="$REGION" \
        --repo-name="$GITHUB_REPO" \
        --repo-owner="$GITHUB_OWNER" \
        --branch-pattern="^staging$" \
        --build-config="cloudbuild-staging.yaml" \
        --description="Deploy to staging on staging branch push" \
        --substitutions="_REGION=$REGION" || echo "⚠️  Trigger already exists or failed"

    echo "✅ Triggers created"
    echo ""
fi

# Step 5: Grant Cloud Build permissions
echo "🔧 Step 5: Granting Cloud Build permissions..."

PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

# Grant Cloud Build permission to deploy to Cloud Run
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/run.admin" \
    --condition=None || true

# Grant permission to act as service accounts
gcloud iam service-accounts add-iam-policy-binding "$API_SA" \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/iam.serviceAccountUser" || true

gcloud iam service-accounts add-iam-policy-binding "$WORKER_SA" \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/iam.serviceAccountUser" || true

echo "✅ Permissions granted"
echo ""

# Step 6: Verify setup
echo "🔧 Step 6: Verifying setup..."

echo "Checking Artifact Registry..."
gcloud artifacts repositories describe hauslet \
    --location="$REGION" \
    --format="value(name)" > /dev/null && echo "✅ Artifact Registry ready"

echo "Checking service accounts..."
gcloud iam service-accounts describe "$API_SA" \
    --format="value(email)" > /dev/null && echo "✅ API service account ready"

gcloud iam service-accounts describe "$WORKER_SA" \
    --format="value(email)" > /dev/null && echo "✅ Worker service account ready"

echo ""

# Step 7: First deployment instructions
echo "📋 Next Steps:"
echo ""
echo "1. Configure secrets in Secret Manager:"
echo "   ./deploy/configure-secrets.sh"
echo ""
echo "2. Deploy infrastructure (Cloud SQL, Redis, VPC):"
echo "   cd deploy/terraform && terraform apply"
echo ""
echo "3. Trigger first deployment:"
echo "   git push origin main"
echo "   # Or manually:"
echo "   gcloud builds submit --config=cloudbuild.yaml --region=$REGION"
echo ""
echo "4. Verify deployment:"
echo "   gcloud run services list --region=$REGION"
echo ""
echo "✨ CI/CD Setup Complete!"
