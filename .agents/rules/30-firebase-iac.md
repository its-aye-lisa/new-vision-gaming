---
trigger: glob
globs: "firebase/**, infra/**, Dockerfile"
description: "Firebase Hosting/Cloud Run/Firestore fail-closed IaC and DISA Container STIG hardening rules."
---

# Firebase & Infrastructure-as-Code Rules

1. **Fail-Closed Firestore:** `firebase/firestore.rules` MUST deny all client writes by default (`allow write: if false;`) and keep sensitive/room-secret documents unreadable directly by clients. All authoritative game-state transitions run in Cloud Run (`nvg-arcade`).
2. **Cloud Run & Hosting Compatibility:** `nvg-arcade` MUST listen on `$PORT` (default `8080`), serve fast cold-start responses (`TTFB < 100ms`), and route dynamic paths through `firebase/firebase.json` rewrites while keeping `/ads.txt`, `/robots.txt`, and `/sitemap.xml` reachable.
3. **DISA Container STIG:** `Dockerfile` MUST remain `FROM scratch`, static `CGO_ENABLED=0`, `GOFIPS140=v1.0.0`, `USER 65532:65532`, `GODEBUG=fips140=on`, and `infra/k8s/` MUST preserve `readOnlyRootFilesystem: true`, `drop: [ALL]`, and `RuntimeDefault` seccomp.
