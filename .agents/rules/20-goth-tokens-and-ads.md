---
trigger: glob
globs: "app/web/**, spec/design-tokens.json, app/cmd/nvg-arcade/**"
description: "GoTH (Go html/template + Tailwind v4 W3C DTCG tokens + Vendored HTMX), Zero-CLS Ad Optimization, Nonce Strict CSP, and A2A/A2UI rules."
---

# GoTH UI, W3C Design Tokens, Ad-Yield & Agent Protocol Rules

1. **W3C DTCG Token-First Styling:** Edit semantic tokens in `spec/design-tokens.json` and `app/web/styles/input.css` (`@theme`). Never add inline `style=""` attributes or arbitrary hex colors in `.tmpl` files. Recompile `app/web/static/tailwind.css` via the standalone Tailwind v4 CLI (`--minify`, zero `node_modules`).
2. **Vendored HTMX Only (Zero CDN JS):** Always serve HTMX from embedded `/static/htmx.min.js` (pinned v2.0.4) with `htmx-config` setting `allowEval: false`, `allowScriptTags: false`, `selfRequestsOnly: true`.
3. **Zero-CLS Ad Monetization & Web-vs-Native Separation (`CLS <= 0.01`):**
   - Every ad container (`nvg-top-leaderboard`, `nvg-sidebar-rect`, between-hand slots) MUST have fixed/reserved `min-h-[...]` dimensions so ad fill never shifts gameplay boards.
   - Keep `ADS_ENABLED=false` strict zero-inline CSP (`script-src 'self'`) as the default; when `ADS_ENABLED=true` on `web`/`pwa`, generate a fresh CSPRNG per-request CSP nonce (`'nonce-...' 'strict-dynamic'`).
   - **Never render Web AdSense inside Native Mobile Shells (`X-NVG-Client: native-ios|native-android`):** `.WebAdsEnabled` MUST evaluate `false` and CSP MUST revert to strict `script-src 'self'` so Phase 2 Capacitor apps use native Google AdMob SDK cleanly without AdSense/Apple Guideline 4.2 policy violations.
4. **PWA, Safe-Areas & Deep-Link Stability:** Preserve `viewport-fit=cover`, safe-area classes (`pt-safe`, `pb-safe`, `px-safe`), `/manifest.webmanifest`, `/app-ads.txt`, `/.well-known/apple-app-site-association`, `/.well-known/assetlinks.json`, and canonical `/games/{slug}` + `/play/{roomCode}` routes.
5. **FIPS 140-3, STIG & Agent Protocols:**
   - Preserve `http.MaxBytesReader`, `DisallowUnknownFields()`, AU-2 structured JSON audit logs, `--require-fips`, `/.well-known/agent-card.json` (A2A v1.0), `/a2a`, and `/api/a2ui/surface` (A2UI v1.0 declarative JSON only — zero raw HTML/JS execution).
