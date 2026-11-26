#!/usr/bin/env bash
# =============================================================================
# One-Time Bootstrap: Wire Your Firebase/GCP Project to GitHub 1-Click Deploy
# =============================================================================
# Run this once inside Google Cloud Shell (or any terminal with gcloud & gh):
#   ./scripts/bootstrap-firebase-github.sh <YOUR_GCP_PROJECT_ID> [ca-pub-XXXX]
#
# What it does automatically:
#   1. Enables Cloud Run, Cloud Build, Artifact Registry, Firebase Hosting & Firestore APIs
#   2. Creates a least-privilege deployer service account (nvg-github-deployer)
#   3. Configures FIREBASE_PROJECT_ID & GCP_SA_KEY in your GitHub repository
#   4. Triggers the 1-Click "Deploy to Firebase & Cloud Run" GitHub Actions workflow!
# =============================================================================
set -euo pipefail

PROJECT_ID="${1:-}"
ADSENSE_ID="${2:-ca-pub-0000000000000000}"
REPO="${GITHUB_REPO:-its-aye-lisa/new-vision-gaming}"
SA_NAME="nvg-github-deployer"

if [ -z "$PROJECT_ID" ]; then
  echo "Usage: $0 <GCP_OR_FIREBASE_PROJECT_ID> [ADSENSE_CLIENT_ID]"
  exit 1
fi

SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "==> [1/5] Enabling required Google Cloud & Firebase APIs on ${PROJECT_ID}..."
gcloud config set project "$PROJECT_ID"
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  firebase.googleapis.com \
  firebasehosting.googleapis.com \
  firestore.googleapis.com

echo "==> [2/5] Creating least-privilege deployer service account (${SA_EMAIL})..."
if ! gcloud iam service-accounts describe "$SA_EMAIL" >/dev/null 2>&1; then
  gcloud iam service-accounts create "$SA_NAME" \
    --display-name="New Vision Gaming GitHub 1-Click Deployer"
fi

for role in \
  roles/run.admin \
  roles/cloudbuild.builds.editor \
  roles/artifactregistry.writer \
  roles/iam.serviceAccountUser \
  roles/firebasehosting.admin \
  roles/firebaserules.admin \
  roles/serviceusage.serviceUsageConsumer; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="$role" \
    --quiet >/dev/null
done

echo "==> [3/5] Generating ephemeral key & uploading to GitHub Secrets (${REPO})..."
TMP_KEY="$(mktemp)"
trap 'rm -f "$TMP_KEY"' EXIT
gcloud iam service-accounts keys create "$TMP_KEY" --iam-account="$SA_EMAIL"

gh variable set FIREBASE_PROJECT_ID --repo "$REPO" --body "$PROJECT_ID"
gh variable set ADSENSE_CLIENT_ID --repo "$REPO" --body "$ADSENSE_ID"
gh secret set GCP_SA_KEY --repo "$REPO" < "$TMP_KEY"
rm -f "$TMP_KEY"

echo "==> [4/5] Triggering 1-Click Deploy to Firebase & Cloud Run..."
gh workflow run "deploy-firebase-cloudrun.yml" \
  --repo "$REPO" \
  -f environment=staging \
  -f ads_enabled=false \
  -f run_full_factory=true

echo "==> [5/5] Done! Watch your live deploy here:"
echo "    https://github.com/${REPO}/actions/workflows/deploy-firebase-cloudrun.yml"
echo "    Live URL (once finished): https://${PROJECT_ID}.web.app"
