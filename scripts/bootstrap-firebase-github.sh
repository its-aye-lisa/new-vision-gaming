#!/usr/bin/env bash
# =============================================================================
# One-Command Bootstrap: Create (or Use) GCP/Firebase Project + Wire GitHub 1-Click Deploy
# =============================================================================
# Run inside Google Cloud Shell (or any terminal with gcloud + gh):
#   ./scripts/bootstrap-firebase-github.sh [PROJECT_ID] [ADSENSE_CLIENT_ID]
#
# If PROJECT_ID is omitted, generates a unique ID like new-vision-gaming-4821
# and creates a brand-new Google Cloud + Firebase project automatically!
# =============================================================================
set -euo pipefail

SUFFIX="$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 5 || echo "prod1")"
PROJECT_ID="${1:-new-vision-gaming-${SUFFIX}}"
ADSENSE_ID="${2:-ca-pub-0000000000000000}"
REPO="${GITHUB_REPO:-its-aye-lisa/new-vision-gaming}"
SA_NAME="nvg-github-deployer"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "==> [1/6] Checking Google Cloud project '${PROJECT_ID}'..."
if ! gcloud projects describe "$PROJECT_ID" >/dev/null 2>&1; then
  echo "    Project does not exist yet — creating brand-new Google Cloud project '${PROJECT_ID}'..."
  gcloud projects create "$PROJECT_ID" --name="New Vision Gaming"
  echo "    Created '${PROJECT_ID}'!"
  echo "    Linking billing account (required for Cloud Run free tier)..."
  BILLING_ID="$(gcloud billing accounts list --filter='open=true' --format='value(name)' | head -n1 || true)"
  if [ -n "$BILLING_ID" ]; then
    gcloud billing projects link "$PROJECT_ID" --billing-account="$BILLING_ID"
    echo "    Linked billing account: ${BILLING_ID}"
  else
    echo "    NOTE: No active billing account auto-detected. Link one at:"
    echo "          https://console.cloud.google.com/billing/linkedaccount?project=${PROJECT_ID}"
  fi
fi

gcloud config set project "$PROJECT_ID"

echo "==> [2/6] Enabling Cloud Run, Artifact Registry, Firebase Hosting & Firestore APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  firebase.googleapis.com \
  firebasehosting.googleapis.com \
  firestore.googleapis.com

echo "==> [3/6] Adding Firebase to project '${PROJECT_ID}'..."
if command -v firebase >/dev/null 2>&1; then
  firebase projects:addfirebase "$PROJECT_ID" 2>/dev/null || true
else
  npx -y firebase-tools@latest projects:addfirebase "$PROJECT_ID" 2>/dev/null || true
fi

echo "==> [4/6] Creating least-privilege GitHub deployer service account (${SA_EMAIL})..."
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

echo "==> [5/6] Uploading FIREBASE_PROJECT_ID & GCP_SA_KEY to GitHub (${REPO})..."
if ! gh auth status >/dev/null 2>&1; then
  echo "    Logging into GitHub CLI once so we can wire GitHub Actions secrets..."
  gh auth login --web
fi

TMP_KEY="$(mktemp)"
trap 'rm -f "$TMP_KEY"' EXIT
gcloud iam service-accounts keys create "$TMP_KEY" --iam-account="$SA_EMAIL"

gh variable set FIREBASE_PROJECT_ID --repo "$REPO" --body "$PROJECT_ID"
gh variable set ADSENSE_CLIENT_ID --repo "$REPO" --body "$ADSENSE_ID"
gh secret set GCP_SA_KEY --repo "$REPO" < "$TMP_KEY"
rm -f "$TMP_KEY"

echo "==> [6/6] Triggering 1-Click Deploy to Firebase & Cloud Run..."
gh workflow run "deploy-firebase-cloudrun.yml" \
  --repo "$REPO" \
  -f environment=staging \
  -f ads_enabled=false \
  -f run_full_factory=true

echo ""
echo "========================================================================="
echo "  ALL DONE! Your new project '${PROJECT_ID}' is wired for 1-Click Deploy!"
echo "  Watch live deploy : https://github.com/${REPO}/actions/workflows/deploy-firebase-cloudrun.yml"
echo "  Live Web/PWA URL  : https://${PROJECT_ID}.web.app"
echo "  Firebase Console  : https://console.firebase.google.com/project/${PROJECT_ID}"
echo "========================================================================="
