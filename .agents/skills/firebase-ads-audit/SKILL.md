---
name: firebase-ads-audit
description: Audit New Vision Gaming for Firebase Hosting/Cloud Run/Firestore fail-closed rules, zero-CLS AdSense/H5 Games Ads SafeFrames, nonce-based strict CSP, /ads.txt, /privacy, and Core Web Vitals compliance.
---

# Skill: Firebase, Ad-Yield & Core Web Vitals Audit (`/firebase-ads-audit`)

1. **Zero-CLS Ad Slot Check:**
   - Inspect `app/web/templates/layout.tmpl` and game templates to verify every ad container (`nvg-top-leaderboard`, `nvg-sidebar-rect`, between-hand banner) reserves fixed `min-h-[...]` space so `CLS <= 0.01`.
   - Confirm no interactive game button sits within accidental-click proximity of an ad container.
2. **Dual-Mode Strict CSP Verification:**
   - Default (`ADS_ENABLED=false`): `Content-Security-Policy` MUST enforce `script-src 'self'` with zero inline scripts/styles.
   - Ad Mode (`ADS_ENABLED=true`): `Content-Security-Policy` MUST include a fresh CSPRNG per-request `'nonce-...' 'strict-dynamic'` matching the `<script nonce="...">` tag.
3. **AdSense & SEO Compliance Endpoints:**
   - Verify `/ads.txt`, `/privacy`, `/robots.txt`, and `/sitemap.xml` return HTTP `200` and stay synchronized between `app/cmd/nvg-arcade/main.go` and `firebase/public/`.
4. **Firebase Fail-Closed IaC Check:**
   - Inspect `firebase/firestore.rules` (default deny client writes), `firebase/firebase.json` (Hosting -> Cloud Run `nvg-arcade` rewrite), and `Dockerfile` (`$PORT`, FIPS 140-3, non-root `65532:65532`).
5. **Run Live Verification:**
   - Run `make dast` and optionally smoke-test `nvg-arcade` with `GODEBUG=fips140=on ADS_ENABLED=true`.
