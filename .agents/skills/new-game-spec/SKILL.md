---
name: new-game-spec
description: Scaffold and verify a new classic game (such as Mexican Train Dominoes) in New Vision Gaming using strict spec-first, pure Go kernel, Clean-Room C verifier, GoTH token UI, and zero-CLS ad-slot engineering.
---

# Skill: Spec-First New Classic Game Launch (`/new-game-spec`)

Use this skill whenever adding a new classic game from `spec/suite.spec.json` (starting with **Mexican Train Dominoes**):

1. **Design Interview (Optional):** Recommend `/grill-me` first if house rules, scoring variants (e.g., Double-9 vs Double-12), or multiplayer room lifecycle have open choices.
2. **Author Formal Specification (`spec/<game>.spec.json`):**
   - Define deterministic board/hand state, legal actions, turn progression, win/scoring rules, formal invariants (`<GAME>-INV-1..N`), and at least 10 acceptance vectors (including an adversarial anti-backdoor probe).
   - Register the game slug, status, session target, and paths in `spec/suite.spec.json` and `spec/openapi.yaml`.
3. **Author Independent Clean-Room Verifier (`cleanroom/c-verifier/<game>_kernel.c`):**
   - Implement the pure rules in ANSI C with zero warnings under both `gcc -Wall -Wextra -Werror` and `clang -Wall -Wextra -Werror`.
4. **Implement Pure AST-Confined Go Kernel (`app/pkg/<game>/`):**
   - Zero forbidden imports, zero `init()`, zero mutable globals, zero `unsafe`/`"C"`.
   - Write comprehensive invariant and property tests in `app/pkg/<game>/*_test.go`.
5. **Wire GoTH + A2A/A2UI Surface (`app/web/` & `app/cmd/nvg-arcade/`):**
   - Create `.tmpl` views styled strictly with semantic tokens from `spec/design-tokens.json` / `app/web/styles/input.css`, then recompile `app/web/static/tailwind.css` via standalone Tailwind v4 CLI.
   - Place ad impressions strictly in reserved zero-CLS slots or natural between-hand breaks (never mid-move).
   - Expose game actions over HTMX (`/hx/...`), REST (`/api/...`), A2A v1.0 (`/a2a`), and A2UI v1.0 (`/api/a2ui/surface`).
6. **Verify via Cost-Efficient Ladder:**
   - Run `make fast-check` (< 1s) -> `make dast` (~2s) -> `make factory && make admit` (~70s).
