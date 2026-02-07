#!/bin/bash
set -e

# Secret Manager Configuration Script for Hauslet
# Creates and configures all required secrets for production and staging

PROJECT_ID="${PROJECT_ID:-}"
ENV="${ENV:-production}"

echo "🔐 Hauslet Secret Manager Configuration"
echo "======================================="
echo ""

if [ -z "$PROJECT_ID" ]; then
    echo "❌ ERROR: PROJECT_ID not set"
    echo "Usage: PROJECT_ID=hauslet-prod ENV=production ./configure-secrets.sh"
    exit 1
fi

gcloud config set project "$PROJECT_ID"

echo "📋 Configuration:"
echo "   Project: $PROJECT_ID"
echo "   Environment: $ENV"
echo ""

# Secret names
SECRETS=(
    "DB_PASSWORD"
    "JWT_SECRET"
    "PAYSTACK_SECRET_KEY"
    "FLUTTERWAVE_WEBHOOK_SECRET"
    "FLUTTERWAVE_OAUTH_CLIENT_ID"
    "FLUTTERWAVE_OAUTH_SECRET"
    "GEMINI_API_KEY"
    "ANTHROPIC_API_KEY"
    "RESEND_API_KEY"
    "R2_SECRET_ACCESS_KEY"
    "R2_ACCESS_KEY_ID"
    "OAUTH_GOOGLE_CLIENT_SECRET"
    "ENCRYPTION_KEY"
    "REDIS_HOST"
)

echo "This script will create/update the following secrets:"
for secret in "${SECRETS[@]}"; do
    echo "  - $secret"
done
echo ""

read -p "Continue? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Cancelled"
    exit 0
fi

echo ""

# Function to create or update secret
create_or_update_secret() {
    local secret_name=$1
    local secret_value=$2

    if gcloud secrets describe "$secret_name" &>/dev/null; then
        echo "📝 Updating existing secret: $secret_name"
        echo -n "$secret_value" | gcloud secrets versions add "$secret_name" --data-file=-
    else
        echo "🆕 Creating new secret: $secret_name"
        echo -n "$secret_value" | gcloud secrets create "$secret_name" --data-file=-
    fi
}

# Prompt for each secret
echo "Enter values for each secret (or press Enter to skip):"
echo ""

for secret in "${SECRETS[@]}"; do
    # Check if secret already exists
    if gcloud secrets describe "$secret" &>/dev/null; then
        read -p "✅ $secret (exists - press Enter to keep, or enter new value): " -s value
    else
        read -p "🔑 $secret: " -s value
    fi
    echo ""

    if [ -n "$value" ]; then
        create_or_update_secret "$secret" "$value"
        echo "✅ $secret configured"
    else
        if gcloud secrets describe "$secret" &>/dev/null; then
            echo "⏭️  $secret unchanged"
        else
            echo "⚠️  $secret skipped (not created)"
        fi
    fi
    echo ""
done

# Grant service accounts access to secrets
echo ""
echo "🔧 Granting service account access to secrets..."

API_SA="hauslet-api-sa@${PROJECT_ID}.iam.gserviceaccount.com"
WORKER_SA="hauslet-worker-sa@${PROJECT_ID}.iam.gserviceaccount.com"

if [ "$ENV" = "staging" ]; then
    API_SA="hauslet-api-staging-sa@${PROJECT_ID}.iam.gserviceaccount.com"
    WORKER_SA="hauslet-worker-staging-sa@${PROJECT_ID}.iam.gserviceaccount.com"
fi

for secret in "${SECRETS[@]}"; do
    if gcloud secrets describe "$secret" &>/dev/null; then
        # Grant API access
        gcloud secrets add-iam-policy-binding "$secret" \
            --member="serviceAccount:$API_SA" \
            --role="roles/secretmanager.secretAccessor" \
            --condition=None &>/dev/null || true

        # Grant Worker access
        gcloud secrets add-iam-policy-binding "$secret" \
            --member="serviceAccount:$WORKER_SA" \
            --role="roles/secretmanager.secretAccessor" \
            --condition=None &>/dev/null || true

        echo "✅ $secret - granted access to service accounts"
    fi
done

echo ""
echo "✨ Secret configuration complete!"
echo ""
echo "Verify secrets:"
echo "  gcloud secrets list"
echo ""
echo "View secret value:"
echo "  gcloud secrets versions access latest --secret=DB_PASSWORD"
