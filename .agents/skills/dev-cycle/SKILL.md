---
name: dev-cycle
description: Cost-efficient spec-driven development cycle for New Vision Gaming (spec -> clean-room C kernel -> pure Go kernel -> Tailwind v4 GoTH UI -> fast-check -> decoupled verifiable factory).
---

# Skill: Spec-Driven Secure Development Cycle (`/dev-cycle`)

1. Read the target GitHub Intent Issue or requirement and delegate spec updates to `spec-architect` (or edit `spec/<game>.spec.json` directly) **before** touching `app/`.
2. Update the independent Clean-Room C reference kernel (`cleanroom/c-verifier/`).
3. Update the pure AST-confined Go kernel (`app/pkg/<game>/`) and exhaustive/property tests.
4. If UI templates (`app/web/templates/*.tmpl`) or design tokens (`spec/design-tokens.json`, `app/web/styles/input.css`) changed, recompile `app/web/static/tailwind.css` via Tailwind v4 CLI (`--minify`).
5. Run Tier-1 Fast Preflight (< 1s):
   ```bash
   make fast-check
   ```
6. Fix any AST confinement, memory-safety, Trojan Source, `go vet`, or invariant failure immediately.
7. Run Tier-3 Live DAST (~2s) if `app/cmd/nvg-arcade/` changed (`make dast`).
8. Once ready for PR/release, run the decoupled 9-gate Verifiable Software Factory & Gatekeeper:
   ```bash
   make factory && make admit
   ```
9. Report the verified binary SHA-256, Clean-Room bit-for-bit match status, Go/GCC/Clang equivalence count, and DAST/FIPS 140-3 result.
