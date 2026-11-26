# Create New Google Cloud / Firebase Project & Deploy (1-Click Walkthrough)

<walkthrough-tutorial-duration duration="3"></walkthrough-tutorial-duration>

Welcome to **New Vision Gaming**! This Cloud Shell walkthrough creates a **brand-new Google Cloud + Firebase project**, enables Cloud Run + Firebase Hosting + Firestore, wires your GitHub Actions 1-Click Deploy button, and deploys **Mexican Train Dominoes** & **Tic-Tac-Toe** live on HTTPS.

## Step 1: Create Your New Project & Wire GitHub 1-Click Deploy

Run this single command (it auto-creates a brand-new `new-vision-gaming-<id>` GCP + Firebase project, links billing, uploads the GitHub deployer secret, and triggers your first live deploy):

```bash
chmod +x scripts/bootstrap-firebase-github.sh
./scripts/bootstrap-firebase-github.sh
```

*(Want a custom project ID? Pass it as the first argument, e.g. `./scripts/bootstrap-firebase-github.sh new-vision-gaming-prod`.)*

## Step 2: Watch Your Live Deploy

Once the script finishes, open your GitHub Actions tab and watch **`Deploy to Firebase & Cloud Run (1-Click)`** verify through the 9-Gate Factory and publish your live URL (`https://<your-project-id>.web.app`).

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

From now on, every deployment is literally **one click** in GitHub:
**`Actions` → `Deploy to Firebase & Cloud Run (1-Click)` → `Run workflow`**!
