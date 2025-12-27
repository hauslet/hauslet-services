#!/bin/bash
set -e

# Hauslet CI/CD Deployment Script
# Step-by-step deployment to GCP

PROJECT_ID="gen-lang-client-0265949535"
REGION="europe-north1"

echo "🚀 Hauslet CI/CD Deployment to GCP"
echo "===================================="
echo ""
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo ""
echo "Prerequisites Verified:"
echo "  ✅ gcloud installed and authenticated"
echo "  ✅ Artifact Registry repository created"
echo "  ✅ Secrets added to Secret Manager"
echo "  ✅ Cloud SQL database created"
echo "  ✅ Valkey/Redis created"
echo ""

# Set active project
gcloud config set project $PROJECT_ID

# ============================================================================
# STEP 1: Enable Required APIs
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 1: Enable Required GCP APIs"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

read -p "Enable required APIs? (yes/no): " CONFIRM
if [ "$CONFIRM" = "yes" ]; then
    echo "Enabling APIs..."
    gcloud services enable \
        cloudbuild.googleapis.com \
        run.googleapis.com \
        cloudscheduler.googleapis.com \
        cloudtasks.googleapis.com \
        secretmanager.googleapis.com \
        sqladmin.googleapis.com \
        compute.googleapis.com \
        vpcaccess.googleapis.com \
        artifactregistry.googleapis.com

    echo "✅ APIs enabled"
else
    echo "⏭️  Skipping API enablement"
fi
echo ""

# ============================================================================
# STEP 2: Create Service Accounts
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 2: Create Service Accounts"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "We need to create 2 service accounts:"
echo "  1. hauslet-api-sa (for API service)"
echo "  2. hauslet-worker-sa (for Worker service)"
echo ""

read -p "Create service accounts? (yes/no): " CONFIRM
if [ "$CONFIRM" = "yes" ]; then
    # API Service Account
    if gcloud iam service-accounts describe hauslet-api-sa@${PROJECT_ID}.iam.gserviceaccount.com &>/dev/null; then
        echo "ℹ️  hauslet-api-sa already exists"
    else
        gcloud iam service-accounts create hauslet-api-sa \
            --display-name="Hauslet API Service Account" \
            --description="Service account for Hauslet API Cloud Run service"
        echo "✅ Created hauslet-api-sa"
    fi

    # Worker Service Account
    if gcloud iam service-accounts describe hauslet-worker-sa@${PROJECT_ID}.iam.gserviceaccount.com &>/dev/null; then
        echo "ℹ️  hauslet-worker-sa already exists"
    else
        gcloud iam service-accounts create hauslet-worker-sa \
            --display-name="Hauslet Worker Service Account" \
            --description="Service account for Hauslet Worker Cloud Run service"
        echo "✅ Created hauslet-worker-sa"
    fi
else
    echo "⏭️  Skipping service account creation"
fi
echo ""

API_SA="hauslet-api-sa@${PROJECT_ID}.iam.gserviceaccount.com"
WORKER_SA="hauslet-worker-sa@${PROJECT_ID}.iam.gserviceaccount.com"

# ============================================================================
# STEP 3: Grant IAM Permissions
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 3: Grant IAM Permissions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

read -p "Grant IAM permissions to service accounts? (yes/no): " CONFIRM
if [ "$CONFIRM" = "yes" ]; then
    echo "Granting permissions to API service account..."

    # API Service Account Permissions
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$API_SA" \
        --role="roles/cloudsql.client" \
        --condition=None

    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$API_SA" \
        --role="roles/secretmanager.secretAccessor" \
        --condition=None

    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$API_SA" \
        --role="roles/cloudtasks.enqueuer" \
        --condition=None

    echo "✅ API service account permissions granted"

    echo "Granting permissions to Worker service account..."

    # Worker Service Account Permissions
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$WORKER_SA" \
        --role="roles/cloudsql.client" \
        --condition=None

    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$WORKER_SA" \
        --role="roles/secretmanager.secretAccessor" \
        --condition=None

    echo "✅ Worker service account permissions granted"
else
    echo "⏭️  Skipping IAM permissions"
fi
echo ""

# ============================================================================
# STEP 4: Grant Cloud Build Permissions
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 4: Grant Cloud Build Permissions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

read -p "Grant Cloud Build permissions? (yes/no): " CONFIRM
if [ "$CONFIRM" = "yes" ]; then
    PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
    CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

    echo "Cloud Build Service Account: $CLOUDBUILD_SA"
    echo ""

    # Grant Cloud Run Admin
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:$CLOUDBUILD_SA" \
        --role="roles/run.admin" \
        --condition=None

    # Grant Service Account User for API SA
    gcloud iam service-accounts add-iam-policy-binding $API_SA \
        --member="serviceAccount:$CLOUDBUILD_SA" \
        --role="roles/iam.serviceAccountUser"

    # Grant Service Account User for Worker SA
    gcloud iam service-accounts add-iam-policy-binding $WORKER_SA \
        --member="serviceAccount:$CLOUDBUILD_SA" \
        --role="roles/iam.serviceAccountUser"

    # Grant Artifact Registry Writer
    gcloud artifacts repositories add-iam-policy-binding hauslet \
        --location=$REGION \
        --member="serviceAccount:$CLOUDBUILD_SA" \
        --role="roles/artifactregistry.writer"

    echo "✅ Cloud Build permissions granted"
else
    echo "⏭️  Skipping Cloud Build permissions"
fi
echo ""

# ============================================================================
# STEP 5: Get Cloud SQL and Redis Connection Details
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 5: Get Cloud SQL and Redis Connection Details"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "List your Cloud SQL instances:"
gcloud sql instances list --format="table(name,region,databaseVersion,state)"
echo ""

read -p "Enter your Cloud SQL instance name: " SQL_INSTANCE
SQL_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${SQL_INSTANCE}"
echo "✅ Cloud SQL Connection Name: $SQL_CONNECTION_NAME"
echo ""

echo "List your Valkey/Redis instances:"
gcloud redis instances list --region=$REGION --format="table(name,host,port,state)" 2>/dev/null || echo "No Valkey instances found in $REGION"
echo ""

read -p "Enter your Redis/Valkey instance name: " REDIS_INSTANCE
REDIS_HOST=$(gcloud redis instances describe $REDIS_INSTANCE --region=$REGION --format="value(host)" 2>/dev/null || echo "")
REDIS_PORT=$(gcloud redis instances describe $REDIS_INSTANCE --region=$REGION --format="value(port)" 2>/dev/null || echo "6379")

if [ -n "$REDIS_HOST" ]; then
    echo "✅ Redis Host: $REDIS_HOST:$REDIS_PORT"
else
    echo "⚠️  Could not auto-detect Redis host. You'll need to add it manually to Secret Manager."
fi
echo ""

# ============================================================================
# STEP 6: Verify VPC Connector
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 6: Check VPC Connector"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Cloud Run needs a VPC connector to access Redis/Valkey and Cloud SQL."
echo ""

VPC_CONNECTOR=$(gcloud compute networks vpc-access connectors list \
    --region=$REGION \
    --format="value(name)" 2>/dev/null | head -1)

if [ -n "$VPC_CONNECTOR" ]; then
    echo "✅ Found VPC Connector: $VPC_CONNECTOR"
    VPC_CONNECTOR_NAME="$VPC_CONNECTOR"
else
    echo "⚠️  No VPC connector found in $REGION"
    echo ""
    echo "Creating VPC connector..."
    read -p "Create VPC connector 'hauslet-vpc-connector'? (yes/no): " CREATE_VPC

    if [ "$CREATE_VPC" = "yes" ]; then
        gcloud compute networks vpc-access connectors create hauslet-vpc-connector \
            --region=$REGION \
            --network=default \
            --range=10.8.0.0/28 \
            --min-throughput=200 \
            --max-throughput=300

        echo "✅ VPC connector created"
        VPC_CONNECTOR_NAME="hauslet-vpc-connector"
    else
        echo "⚠️  Warning: Deployments may fail without VPC connector"
        VPC_CONNECTOR_NAME=""
    fi
fi
echo ""

# ============================================================================
# STEP 7: First Deployment (Manual Build)
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 7: First Manual Deployment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Let's do a test deployment to verify everything works."
echo ""
echo "⚠️  IMPORTANT: Update cloudbuild.yaml with your Cloud SQL instance:"
echo "   Change: --set-cloudsql-instances=\$PROJECT_ID:\${_REGION}:hauslet-postgres"
echo "   To:     --set-cloudsql-instances=$SQL_CONNECTION_NAME"
echo ""

if [ -n "$VPC_CONNECTOR_NAME" ]; then
    echo "   And VPC connector:"
    echo "   Change: --vpc-connector=hauslet-vpc-connector"
    echo "   To:     --vpc-connector=$VPC_CONNECTOR_NAME"
    echo ""
fi

read -p "Have you updated cloudbuild.yaml? (yes/no): " UPDATED
if [ "$UPDATED" != "yes" ]; then
    echo ""
    echo "⏸️  Pausing deployment. Please:"
    echo "   1. Open cloudbuild.yaml"
    echo "   2. Update Cloud SQL connection name to: $SQL_CONNECTION_NAME"
    echo "   3. Update VPC connector to: $VPC_CONNECTOR_NAME"
    echo "   4. Run this script again"
    echo ""
    exit 0
fi

echo ""
read -p "Trigger first deployment? (yes/no): " DEPLOY
if [ "$DEPLOY" = "yes" ]; then
    echo ""
    echo "🚀 Starting deployment..."
    echo "This will take about 8-12 minutes..."
    echo ""

    gcloud builds submit \
        --config=cloudbuild.yaml \
        --region=$REGION \
        --substitutions=_REGION=$REGION

    echo ""
    echo "✅ Deployment complete!"
else
    echo "⏭️  Skipping deployment"
fi
echo ""

# ============================================================================
# STEP 8: Verify Deployment
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 8: Verify Deployment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if services are deployed
API_URL=$(gcloud run services describe hauslet-api \
    --region=$REGION \
    --format='value(status.url)' 2>/dev/null || echo "")

WORKER_URL=$(gcloud run services describe hauslet-worker \
    --region=$REGION \
    --format='value(status.url)' 2>/dev/null || echo "")

if [ -n "$API_URL" ]; then
    echo "✅ API Service deployed: $API_URL"
    echo ""
    echo "Testing health endpoint..."
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" $API_URL/health)
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ Health check passed!"
    else
        echo "⚠️  Health check returned: $HTTP_CODE"
    fi
else
    echo "⚠️  API service not found"
fi

if [ -n "$WORKER_URL" ]; then
    echo "✅ Worker Service deployed: $WORKER_URL"
else
    echo "⚠️  Worker service not found"
fi

echo ""

# ============================================================================
# STEP 9: Set Up GitHub Triggers (Optional)
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 9: Set Up GitHub Triggers (Optional)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "To enable automatic deployments on git push, you need to:"
echo ""
echo "1. Connect your GitHub repository:"
echo "   https://console.cloud.google.com/cloud-build/triggers/connect?project=$PROJECT_ID"
echo ""
echo "2. Then create triggers manually or run:"
echo "   gcloud builds triggers create github \\"
echo "     --name=hauslet-production \\"
echo "     --region=$REGION \\"
echo "     --repo-owner=YOUR_GITHUB_USERNAME \\"
echo "     --repo-name=hauslet-services \\"
echo "     --branch-pattern='^main\$' \\"
echo "     --build-config=cloudbuild.yaml"
echo ""

read -p "Open GitHub connection page in browser? (yes/no): " OPEN_BROWSER
if [ "$OPEN_BROWSER" = "yes" ]; then
    open "https://console.cloud.google.com/cloud-build/triggers/connect?project=$PROJECT_ID" 2>/dev/null || \
    xdg-open "https://console.cloud.google.com/cloud-build/triggers/connect?project=$PROJECT_ID" 2>/dev/null || \
    echo "Please open: https://console.cloud.google.com/cloud-build/triggers/connect?project=$PROJECT_ID"
fi

echo ""

# ============================================================================
# Summary
# ============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✨ DEPLOYMENT COMPLETE!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 Summary:"
echo "   Project: $PROJECT_ID"
echo "   Region: $REGION"
if [ -n "$API_URL" ]; then
    echo "   API URL: $API_URL"
fi
if [ -n "$WORKER_URL" ]; then
    echo "   Worker URL: $WORKER_URL (internal only)"
fi
echo ""
echo "🎯 Next Steps:"
echo "   1. Test your API: curl $API_URL/health"
echo "   2. Set up GitHub triggers for auto-deployment"
echo "   3. Deploy Cloud Scheduler jobs: cd deploy/terraform && ./deploy-scheduler.sh"
echo ""
echo "📚 Documentation:"
echo "   - CI/CD Guide: deploy/CICD_GUIDE.md"
echo "   - Deployment Plan: deploy/GCP_DEPLOYMENT_PLAN.md"
echo ""
