# New Vision Gaming — Classic Games Suite (Firebase + GoTH + A2A/A2UI)

**New Vision Gaming** is an ad-optimized, instant-load, zero-bloat classic games suite on **Firebase Hosting + Cloud Run + Firestore**, shipping live with **Mexican Train Dominoes (Double-9 & Double-12, 2-Player `/play/MEX-XXXX` Invite Rooms + Bot)** and **Tic-Tac-Toe**.

---

## One-Click Deploy to the Web

### Option 1 — Instant Live Web URL (Literally 1 Button, Zero CLI Setup)
[![Run on Google Cloud](https://deploy.cloud.run/button.svg)](https://deploy.cloud.run/?git_repo=https://github.com/its-aye-lisa/new-vision-gaming)

Click **Run on Google Cloud** above → pick your GCP project → get a live HTTPS URL for `nvg-arcade` (FIPS 140-3, `FROM scratch`, scale-to-zero) in ~2 minutes!

### Option 2 — Full Firebase Hosting + Cloud Run + GitHub Actions 1-Click Button
[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://shell.cloud.google.com/cloudshell/open?git_repo=https://github.com/its-aye-lisa/new-vision-gaming&tutorial=docs/ONE_CLICK_DEPLOY.md)

Run the 60-second bootstrap once in Cloud Shell, and from then on deploy any update with **1 click** inside GitHub:
**`Actions` → `Deploy to Firebase & Cloud Run (1-Click)` → `Run workflow`**

---

## Architecture Highlights

- **Decoupled Zero-Trust Software Factory:** Verified, built bit-for-bit in Clean-Room enclaves, 3-way checked (`Go` vs `GCC` vs `Clang`), scanned, and 9/9 Gatekeeper-admitted by [`its-aye-lisa/verifiable-software-factory`](https://github.com/its-aye-lisa/verifiable-software-factory).
- **Live Games:**
  - **Mexican Train Dominoes (`/games/mexican-train` & `/play/MEX-XXXX`):** Double-9 & Double-12, 6-letter friend invite links with isolated `HttpOnly` hidden hands, smart pure-Go bot, natural between-hand zero-CLS ad slot, 3-way Go/GCC/Clang verified.
  - **Tic-Tac-Toe (`/games/tictactoe`):** Exhaustively verified across all 4,520 reachable states.
- **GoTH + W3C Design Tokens + PWA:** Go 1.24 (`CGO_ENABLED=0`, `GOFIPS140=v1.0.0`, zero third-party Go modules) + `html/template` + Tailwind v4 `@theme` (`spec/design-tokens.json`) + vendored HTMX 2.0.4 + installable `/manifest.webmanifest`.
- **Phase 1 → Phase 2 App Store Ready:** Pure 0-import game kernels compile to `GOOS=js GOARCH=wasm` for offline iOS/Android Capacitor play; `X-NVG-Client: native-ios|native-android` automatically suppresses Web AdSense tags for native AdMob SDK; `/app-ads.txt`, Apple Universal Links, and Android App Links pre-wired.
- **AI-Native Protocols:** **A2A v1.0** (`/.well-known/agent-card.json`, `POST /a2a`) + **A2UI v1.0** (`GET /api/a2ui/surface`).

---

## Local Development

```bash
git clone https://github.com/its-aye-lisa/verifiable-software-factory.git
git clone https://github.com/its-aye-lisa/new-vision-gaming.git
cd new-vision-gaming

make fast-check              # <1s AST confinement + invariants + offline WASM check
make factory && make admit   # Full 9-gate FIPS + Clean-Room + DAST + SBOM/VEX/SLSA
make simulate-attacks        # 6 live Untrusted-Developer Red-Team simulations
make serve-ads               # Launch locally with FIPS 140-3 + AdSense Nonce CSP
```
