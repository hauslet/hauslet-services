#!/bin/bash
set -e

# Configuration
PROJECT_ID=$(gcloud config get-value project)
SERVICE_ACCOUNT_NAME="hauslet-ai-support"
DISPLAY_NAME="Hauslet AI Support Service Account"
KEY_FILE="vertex-ai-credentials.json"

echo "==================================================="
echo "   Hauslet Vertex AI Setup (Phase 3)"
echo "==================================================="
echo "Project ID: $PROJECT_ID"
echo "Service Account: $SERVICE_ACCOUNT_NAME"
echo "==================================================="

# 1. Enable Required APIs
echo ""
echo "[1/4] Enabling Required APIs..."
echo "Enabling aiplatform.googleapis.com..."
gcloud services enable aiplatform.googleapis.com
echo "Enabling discoveryengine.googleapis.com (Agent Builder)..."
gcloud services enable discoveryengine.googleapis.com
echo "Enabling dialogflow.googleapis.com (Required for some agent types)..."
gcloud services enable dialogflow.googleapis.com

# 2. Create Service Account
echo ""
echo "[2/4] Checking/Creating Service Account..."
if gcloud iam service-accounts describe "${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" > /dev/null 2>&1; then
    echo "Service account ${SERVICE_ACCOUNT_NAME} already exists."
else
    echo "Creating service account ${SERVICE_ACCOUNT_NAME}..."
    gcloud iam service-accounts create "${SERVICE_ACCOUNT_NAME}" \
        --display-name "${DISPLAY_NAME}"
fi

SA_EMAIL="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# 3. Grant IAM Permissions
echo ""
echo "[3/4] Granting IAM Roles..."
# Vertex AI User (General access)
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/aiplatform.user" > /dev/null

# Discovery Engine Editor (For Agent Builder/Search)
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/discoveryengine.editor" > /dev/null

echo "Roles granted: aiplatform.user, discoveryengine.editor"

# 4. Generate Key File (Optional but needed for local dev)
echo ""
echo "[4/4] Service Account Key"
if [ -f "$KEY_FILE" ]; then
    echo "Key file $KEY_FILE already exists. Skipping generation."
else
    echo "Generating new key file: $KEY_FILE"
    gcloud iam service-accounts keys create "$KEY_FILE" \
        --iam-account="${SA_EMAIL}"
fi

echo ""
echo "==================================================="
echo "Setup Complete!"
echo "==================================================="
echo ""
echo "NEXT STEPS (Console UI):"
echo "1. Go to: https://console.cloud.google.com/gen-app-builder/engines"
echo "2. Click 'Create App'"
echo "3. Select 'Chat' as the type."
echo "4. Create a 'Data Store' (e.g., upload your docs/ folder or point to a website)."
echo "5. Once created, copy the 'Data Store ID' (or Agent ID) and 'Location'."
echo ""
echo "UPDATE YOUR .ENV:"
echo "VERTEX_AI_PROJECT_ID=$PROJECT_ID"
echo "VERTEX_AI_AGENT_ID=<YOUR_NEW_AGENT_ID>"
echo "VERTEX_AI_LOCATION=global (or us-central1)"
echo "VERTEX_AI_CREDENTIALS_PATH=$(pwd)/$KEY_FILE"
echo "==================================================="