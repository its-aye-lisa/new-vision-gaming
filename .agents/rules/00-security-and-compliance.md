---
trigger: always_on
description: Always-active zero-trust, memory-safety, spec-first, and decoupled software-factory guardrails for New Vision Gaming.
---

# Mandatory Security & Compliance Guardrails (New Vision Gaming)

1. **Spec-First Rule:** Any behavioral change to a game MUST start in `spec/<game>.spec.json` (and `spec/suite.spec.json` / `spec/openapi.yaml` when routes change). Never introduce undocumented flags, cheat codes, or actor-dependent bypasses.
2. **Decoupled Factory Boundary:** Never copy `verifiable-software-factory` Go source into this repo. Always invoke `../verifiable-software-factory/bin/factory --target .` via `Makefile`.
3. **Memory Safety (CISA/NSA 2025–2026):**
   - NEVER import `unsafe` or `"C"` anywhere under `app/`.
   - NEVER add `//go:linkname`, `//go:cgo_`, `//go:noescape`, `//go:nosplit`, `.s` assembly, or `.syso` objects under `app/`.
4. **Minimal TCB (Zero External Go Modules):** Do NOT add third-party dependencies to `go.mod`. Use Go 1.24 stdlib (`crypto/fips140`, `crypto/ed25519`, `net/http`, `html/template`, `embed`).
5. **No Gate Weakening or Secret Commits:** Never weaken `policy/factory-policy.json`, `cleanroom/`, `firebase/firestore.rules`, or `.github/workflows/`. Never read, print, or commit `.factory-state/*.key.json`, `.env`, or private keys.
