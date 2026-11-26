# One-Click Web Deployment (Google Cloud Run + Firebase Hosting)

<walkthrough-tutorial-duration duration="3"></walkthrough-tutorial-duration>

Welcome to **New Vision Gaming**! Follow these 2 clicks inside Google Cloud Shell to launch your verified **Mexican Train Dominoes** & **Tic-Tac-Toe** Classic Games Suite live on HTTPS.

## Step 1: Pick Your Google Cloud / Firebase Project

<walkthrough-project-setup billing="true"></walkthrough-project-setup>

## Step 2: Run the 1-Command Auto-Deploy

Click the button below to deploy directly to **Cloud Run (FIPS 140-3)** and wire up your **GitHub Actions 1-Click Deploy** button:

```bash
chmod +x scripts/bootstrap-firebase-github.sh
./scripts/bootstrap-firebase-github.sh "$GOOGLE_CLOUD_PROJECT"
```

Or deploy immediately right from Cloud Shell without GitHub secrets:

```bash
gcloud run deploy nvg-arcade \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --min-instances 0 \
  --set-env-vars GODEBUG=fips140=on,ADS_ENABLED=false
```

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

Once deployed, every future release is literally **one click**:
**GitHub → Actions → Deploy to Firebase & Cloud Run (1-Click) → Run workflow**!
