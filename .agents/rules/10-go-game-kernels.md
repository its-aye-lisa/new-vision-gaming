---
trigger: glob
globs: "app/pkg/**/*.go, cleanroom/**"
description: "Pure AST-confined Go game kernel and independent C Clean-Room verifier invariants."
---

# Pure Game Kernel & Clean-Room Verifier Rules

1. **Zero-Import Pure Core:** Packages under `app/pkg/<game>/` MUST have **0 imports** (or only explicitly policy-approved stdlib packages in `policy/factory-policy.json`), **0 `init()` functions**, and **0 mutable package-level globals**.
2. **Determinism & Exhaustive Invariants:** Every game kernel MUST have deterministic pure functions backed by `*_test.go` property/exhaustive tests proving all invariants declared in `spec/<game>.spec.json`.
3. **Clean-Room 3-Way Equivalence:** Every behavioral change in `spec/<game>.spec.json` MUST be mirrored in `cleanroom/c-verifier/` and compile cleanly with zero warnings under both `gcc -Wall -Wextra -Werror` and `clang -Wall -Wextra -Werror`.
4. **No Hidden State or Cheats:** Never branch game outcomes on player names, actor strings, wall-clock time, or environment variables.
5. **Offline Mobile WASM Portability (Phase 2 App Store):** Every pure game kernel under `app/pkg/<game>/` MUST compile cleanly under `GOOS=js GOARCH=wasm go build ./app/pkg/...` (`make check-wasm`, also enforced inside `make fast-check`) so iOS/Android Capacitor apps can bundle the exact same verified engine offline.
