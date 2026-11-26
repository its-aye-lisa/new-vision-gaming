# New Vision Gaming — Classic Games Suite (Firebase + GoTH + A2A/A2UI)

**New Vision Gaming** is an ad-optimized, instant-load classic games suite deployed on **Firebase Hosting + Cloud Run + Firestore**, starting with **Tic-Tac-Toe** (`LIVE`) and scaling cleanly into Connect Four, Minesweeper, Checkers, Daily Sudoku, and Chess.

This repository contains **ONLY the product application, specs, clean-room game verifiers, and Firebase/IaC configs**. All build, SAST/SCA/DAST, Clean-Room reproducibility, SBOM/VEX/SLSA attestation, and red-team gates are executed by the decoupled reusable factory in [`../verifiable-software-factory`](file:///home/antigravity/verifiable-software-factory).

---

## Highlights
- **GoTH Stack:** Go 1.24 + `html/template` (`.tmpl`) + W3C DTCG Token-Based Tailwind CSS v4 (`spec/design-tokens.json`) + Pinned Vendored HTMX 2.0.4 (`embed.FS`, zero external CDNs).
- **Ad Monetization Ready:** Reserved zero-CLS (`CLS < 0.01`) ad slots (728×90 Leaderboard, 300×250 Right Rail), Google AdSense/GPT **Strict Nonce-Based CSP**, `/ads.txt`, `/privacy`, `/robots.txt`, `/sitemap.xml`.
- **Firebase Production Stack (`firebase/`):** Firebase Hosting rewrites to Cloud Run (`nvg-arcade`), separate `dev`/`staging`/`prod` projects, fail-closed `firestore.rules` preventing client leaderboard tampering.
- **AI-Native Protocols:** **A2A v1.0** (`/.well-known/agent-card.json`, `POST /a2a`) + **A2UI v1.0** (`GET /api/a2ui/surface`).
- **Verifiable & Memory-Safe:** Pure Go (`CGO_ENABLED=0`, `GOFIPS140=v1.0.0`), zero third-party Go modules, AST-confined game kernels, exhaustively verified across all 4,520 reachable Tic-Tac-Toe states.

## Quickstart
```bash
make fast-check          # <1s SAST/AST/Trojan/Memory-Safety + exhaustive invariants
make factory && make admit   # Full 9-gate factory + Clean-Room + DAST + SBOM/VEX/SLSA
make simulate-attacks    # 6 live Untrusted-Developer Red-Team simulations
make serve               # Launch verified arcade on http://127.0.0.1:8080
make serve-ads           # Launch with AdSense Nonce Strict CSP enabled
```
