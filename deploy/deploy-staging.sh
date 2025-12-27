#!/bin/bash
set -e

# Quick Staging Deployment Script

PROJECT_ID="gen-lang-client-0265949535"
REGION="europe-north1"

echo "🧪 Deploying Hauslet to STAGING"
echo "================================"
echo ""
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo ""

gcloud config set project $PROJECT_ID

# Check if VPC connector exists
echo "Checking VPC connector..."
VPC_EXISTS=$(gcloud compute networks vpc-access connectors list \
    --region=$REGION \
    --filter="name:hauslet-vpc-connector" \
    --format="value(name)" 2>/dev/null || echo "")

if [ -z "$VPC_EXISTS" ]; then
    echo "⚠️  VPC connector not found. Creating it..."
    echo "This will take ~2 minutes..."

    gcloud compute networks vpc-access connectors create hauslet-vpc-connector \
        --region=$REGION \
        --network=default \
        --range=10.8.0.0/28 \
        --min-throughput=200 \
        --max-throughput=300

    echo "✅ VPC connector created"
else
    echo "✅ VPC connector exists: $VPC_EXISTS"
fi

echo ""
echo "📦 Configuration:"
echo "   Cloud SQL: hauslet-postgres-primary"
echo "   VPC Connector: hauslet-vpc-connector"
echo "   Services: hauslet-api-staging, hauslet-worker-staging"
echo ""

read -p "Start staging deployment? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Deployment cancelled"
    exit 0
fi

echo ""
echo "🚀 Starting staging deployment..."
echo "This will take ~8-12 minutes..."
echo ""

# Change to project root (parent directory)
cd "$(dirname "$0")/.."

# Submit build
gcloud builds submit \
    --config=cloudbuild-staging.yaml \
    --region=$REGION

echo ""
echo "✅ Staging deployment complete!"
echo ""

# Get staging URLs
API_URL=$(gcloud run services describe hauslet-api-staging \
    --region=$REGION \
    --format='value(status.url)' 2>/dev/null || echo "")

WORKER_URL=$(gcloud run services describe hauslet-worker-staging \
    --region=$REGION \
    --format='value(status.url)' 2>/dev/null || echo "")

if [ -n "$API_URL" ]; then
    echo "📍 Staging API: $API_URL"
    echo ""
    echo "Testing health endpoint..."
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" $API_URL/health)

    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ Health check passed!"
    else
        echo "⚠️  Health check returned: $HTTP_CODE"
        echo ""
        echo "Check logs:"
        echo "gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=hauslet-api-staging' --limit=20"
    fi
fi

if [ -n "$WORKER_URL" ]; then
    echo "📍 Staging Worker: $WORKER_URL (internal only)"
fi

echo ""
echo "🎯 Next Steps:"
echo "   1. Test your staging API: curl $API_URL/health"
echo "   2. If everything works, update production:"
echo "      - Update cloudbuild.yaml with same Cloud SQL instance"
echo "      - Run: gcloud builds submit --config=cloudbuild.yaml --region=$REGION"
echo ""
echo "📊 View deployment:"
echo "   Console: https://console.cloud.google.com/run?project=$PROJECT_ID"
echo ""
