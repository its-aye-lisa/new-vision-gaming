# AGENTS.md — New Vision Gaming (Antigravity 2.0 Contract)

Standalone **New Vision Gaming** Classic Games Suite on **Firebase Hosting + Cloud Run (Go 1.24 FIPS 140-3) + Firestore**, built with **Go `html/template` + W3C DTCG Tailwind CSS v4 + Vendored HTMX 2.0.4 (GoTH)**, **Zero-CLS Ad-Optimized SafeFrames**, and **A2A v1.0 + A2UI v1.0**.

Verified by sibling [`../verifiable-software-factory`](../verifiable-software-factory) and governed commercially by [`../new-vision-gaming-company`](../new-vision-gaming-company).

---

## 1. Repository Map (Read Only What You Need)

| Path | Trust Zone | Purpose |
| :--- | :--- | :--- |
| `spec/` | **Single Source of Truth** | `suite.spec.json` (Tic-Tac-Toe Live, Mexican Train Flagship Next), `<game>.spec.json`, `design-tokens.json`, `openapi.yaml`. |
| `app/pkg/<game>/` | Pure Confined Core | AST-confined pure Go game kernels (0 forbidden imports, 0 `init()`, 0 mutable globals, 0 `unsafe`/`"C"`). |
| `app/web/` | GoTH Token UI | `.tmpl` templates, Tailwind v4 `@theme` (`styles/input.css` -> `static/tailwind.css`), vendored `htmx.min.js` v2.0.4. |
| `app/cmd/nvg-arcade/` | Cloud Run Runtime | `$PORT` server, FIPS 140-3, SI-7 self-attestation, `ADS_ENABLED` CSPRNG CSP nonce, `/ads.txt`, `/privacy`, A2A/A2UI. |
| `cleanroom/c-verifier/` | **Independent IV&V** | Independent ANSI C kernels compiled with both `gcc` and `clang` for 3-way equivalence against Go. |
| `firebase/` & `infra/` | Everything-as-Code | Firebase Hosting rewrites, fail-closed `firestore.rules`, scratch `Dockerfile`, K8s/Kyverno/Terraform IaC. |
| `.agents/` | **Antigravity 2.0 Control Plane** | `rules/` (glob/always_on, incl. `40-bounded-sessions.md`), `skills/` (`/bounded-session`, `/dev-cycle`, `/new-game-spec`, `/firebase-ads-audit`, `/compliance-gate`), `agents/` (custom subagents), `hooks.json` (deterministic `PreToolUse` + `Stop` fast-check gate), `mcp_config.json` (lazy MCP). |

---

## 2. Antigravity 2.0 Custom Subagents & Orchestrator Routing

- **Custom Subagents (`.agents/agents/`):**
  - `spec-architect` (`pro`, `commandExecutionPolicy: off`) -> edits `spec/**` only.
  - `game-kernel-builder` (`inherit`, `sandbox`) -> builds `app/**` pure Go kernels & GoTH views.
  - `cleanroom-ivv-auditor` (`pro`, `sandbox`, read-only) -> independent Clean-Room C/GCC/Clang & FedRAMP/IL4 auditor.
  - `ad-cwv-auditor` (`flash`, `sandbox`) -> zero-CLS ad slots, CSP nonce, `/ads.txt`, Core Web Vitals & Firestore rules audit.
- **Bounded Sessions (`.agents/rules/40-bounded-sessions.md` & `/bounded-session`):**
  - Separate **Explore** (read-only), **Execute** (bounded edits + tests), and **Clean-Eyes Review** (fresh diff-only audit) sessions; gate commits to `.session/daily-plan.md` by verified facts only; stop & reset on the 2-strike circuit breaker.
- **Built-In Antigravity 2.0 Orchestrators:**
  - Use `/grill-me` before authoring a new game spec (`spec/mexican-train.spec.json`) to resolve house-rule & multiplayer decisions.
  - Use `/plan` for multi-file feature plans; `/swarm` for parallel independent tracks (Go kernel + C verifier + `.tmpl` UI); `/boost` for tricky formal invariants; `/teamwork` for full multi-milestone game launches; `/compact` after heavy research.

---

## 3. Verification Ladder & Automated Hooks

`.agents/hooks.json` automatically enforces zero-trust `PreToolUse` denials (secrets, `unsafe`, `"C"`, `//go:linkname`, force-push) and a `Stop` gate running `make fast-check` whenever `app/`, `spec/`, `policy/`, or `cleanroom/` changed.

| Tier | Command | Runtime | When to Use |
| :--- | :--- | :--- | :--- |
| **Tier 1** | `make fast-check` | `< 1s` | After every edit (AST, Trojan Source, memory safety, `go vet`, exhaustive invariants). |
| **Tier 2** | `go test -v ./app/...` | `< 1s` | While iterating on game kernels or handlers. |
| **Tier 3** | `make dast` | `~2s` | After editing `app/cmd/nvg-arcade/`, headers, CSP, templates, or A2A/A2UI. |
| **Tier 4** | `make factory && make admit` | `~70s` | Before PR merge/release (decoupled 9-gate FIPS + Clean-Room + GCC/Clang + SBOM/VEX/SARIF). |
| **Tier 5** | `make simulate-attacks` | `~60s` | ConMon / red-team control regression verification. |
