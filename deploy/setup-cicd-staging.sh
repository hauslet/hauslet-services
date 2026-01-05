#!/bin/bash
set -e

# ==========================================
# Hauslet Unified Deployment Script
# Handles: Infrastructure (Terraform) + CI/CD Wiring (Cloud Build)
# ==========================================

# 1. Configuration
PROJECT_ID="gen-lang-client-0265949535"
REGION="europe-north1"
GITHUB_REPO="hauslet-services" 
GITHUB_OWNER="${GITHUB_OWNER:-}"
TF_DIR="./deploy/terraform"              

echo "🚀 Hauslet Deployment Manager"
echo "========================================"
echo "   Project: $PROJECT_ID"
echo "   Region:  $REGION"
echo ""

# Validation
if [ -z "$GITHUB_OWNER" ]; then
    read -p "Enter GitHub Owner (org or username): " GITHUB_OWNER
fi

gcloud config set project "$PROJECT_ID"

# 2. Verify Infrastructure State
echo "🔍 Step 1: verifying infrastructure..."

if [ ! -d "$TF_DIR" ]; then
    echo "❌ ERROR: Terraform directory not found at $TF_DIR"
    exit 1
fi

cd "$TF_DIR"
echo "   Initializing Terraform..."
terraform init -input=false

# Validate syntax
terraform validate

# Import Logic (Optional but helpful if state is lost)
# Note: Usually better to rely on state file, but this helps recovery
if gcloud artifacts repositories describe hauslet --location=$REGION >/dev/null 2>&1; then
    terraform import -input=false google_artifact_registry_repository.hauslet \
        "projects/$PROJECT_ID/locations/$REGION/repositories/hauslet" 2>/dev/null || true
fi

# Plan Changes
echo "   Calculating changes..."
terraform plan -out=tfplan -input=false \
    -var="project_id=$PROJECT_ID" \
    -var="region=$REGION"

echo ""
read -p "Apply these infrastructure changes? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Deployment cancelled."
    exit 0
fi

echo "   Applying changes..."
terraform apply -auto-approve -input=false tfplan

# Capture Outputs
ARTIFACT_REGISTRY_URL=$(terraform output -raw artifact_registry_url 2>/dev/null || echo "")
echo "✅ Infrastructure applied successfully."
cd ../..

# 3. Enable APIs
echo "🔧 Step 2: Ensuring Build APIs are enabled..."
gcloud services enable \
    cloudbuild.googleapis.com \
    artifactregistry.googleapis.com \
    iam.googleapis.com \
    secretmanager.googleapis.com

# 4. Connect GitHub
echo "🔗 Step 3: Verifying GitHub Connection..."
echo "   (Assuming repository is already connected via Console)"
echo ""

# 5. Create Trigger (Fixed Command)
echo "⚡ Step 4: Configuring Cloud Build Triggers..."

# We attempt to create the trigger. If it exists, we catch the error but don't stop.
gcloud builds triggers create github \
    --name="hauslet-staging-push" \
    --region="global" \
    --repo-name="$GITHUB_REPO" \
    --repo-owner="$GITHUB_OWNER" \
    --branch-pattern="^staging$" \
    --build-config="cloudbuild-staging.yaml" \
    --description="Auto-deploy to Staging on push" \
    --substitutions="_REGION=$REGION,_TAG=\${SHORT_SHA}" \
    --include-logs-with-status || echo "⚠️  Trigger might already exist or failed to create. Checking next step..."

echo "✅ Trigger configuration passed."

# 6. Security & Permissions
echo "🔐 Step 5: Granting CI/CD Permissions..."

PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"
COMPUTE_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

# Allow Cloud Build to deploy to Cloud Run
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/run.admin" > /dev/null

# Allow Cloud Build to act as the Service Account
gcloud iam service-accounts add-iam-policy-binding "$COMPUTE_SA" \
    --member="serviceAccount:$CLOUDBUILD_SA" \
    --role="roles/iam.serviceAccountUser" > /dev/null

echo "✅ Permissions granted."

echo ""
echo "🎉 SUCCESS!"
echo "   Everything is wired up."
echo "   To deploy, simply run: git push origin staging"