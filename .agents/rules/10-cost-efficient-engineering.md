---
trigger: model_decision
description: "Apply whenever planning multi-step edits, optimizing token/context budgets, or deciding which local/CI verification tier to run."
---

# Cost-Efficient Engineering & Token Discipline

1. **Progressive Context Loading:**
   - Start from `AGENTS.md` and read only the files directly needed for the active task.
   - Do NOT bulk-read `dist/` generated JSON artifacts or `app/web/static/htmx.min.js` unless debugging a specific failure.
   - Rely on `glob`-scoped rules and on-demand skills (`/dev-cycle`, `/new-game-spec`, `/firebase-ads-audit`, `/compliance-gate`).
2. **Verification Ladder (Cheapest Sufficient Check First):**
   - Tier 1 (`< 1s`): `make fast-check` after every edit.
   - Tier 2 (`< 1s`): `go test -v ./app/...` while iterating on game logic or handlers.
   - Tier 3 (`~2s`): `make dast` after modifying HTTP routes, CSP/STIG headers, templates, or A2A/A2UI.
   - Tier 4 (`~70s`): `make factory && make admit` ONLY once Tier 1–3 are green and ready for release/PR.
3. **Antigravity 2.0 Multi-Agent & Context Hygiene:**
   - Use `/compact` after large research phases before implementing.
   - Delegate independent tracks (e.g., pure Go kernel vs. Clean-Room C verifier vs. Tailwind `.tmpl` UI) to custom subagents in `.agents/agents/` (`flash` for audits/lookups, `pro` for formal invariants & IV&V).
