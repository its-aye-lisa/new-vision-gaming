---
name: secure-feature-dev
description: Spec-driven, untrusted-developer-resilient feature development workflow for New Vision Gaming (spec -> Clean-Room C verifier -> pure Go kernel -> GoTH/A2A/A2UI -> decoupled factory).
---

# Skill: Spec-Driven Secure Feature Development

Follow this exact sequence whenever implementing a GitHub Intent Issue or modifying `app/`:

1. **Update Specification First (`spec/<game>.spec.json`, `spec/suite.spec.json`, `spec/openapi.yaml`):**
   - Add or update formal invariants and concrete acceptance vectors (including at least one adversarial/backdoor probe vector).
2. **Update Clean-Room Reference Kernel (`cleanroom/c-verifier/`):**
   - Implement the exact spec change in the independent C verifier with zero warnings under `gcc` and `clang` (`-Wall -Wextra -Werror`).
3. **Update Pure Confined Go Kernel (`app/pkg/<game>/`):**
   - Keep the package pure (0 forbidden imports, 0 `init()`, 0 mutable globals, 0 `unsafe`/`"C"`).
   - Update exhaustive state-space / invariant tests in `*_test.go`.
4. **Update Cloud Run / GoTH Adapter (`app/cmd/nvg-arcade/main.go` & `app/web/`) if API/UI changed:**
   - Use W3C DTCG tokens (`spec/design-tokens.json` -> `app/web/styles/input.css` -> `tailwind.css`), vendored HTMX, zero-CLS ad slots, `http.MaxBytesReader`, `DisallowUnknownFields()`, STIG headers, and structured JSON `AU-2` audit logging.
5. **Run Cost-Efficient Verification Ladder:**
   - Step A: `make fast-check` (< 1s)
   - Step B: `make dast` (~2s)
   - Step C: `make factory && make admit` (~70s via `../verifiable-software-factory`)
   - Step D: Summarize spec-to-test traceability in the PR description linking `Closes #<issue>`.
